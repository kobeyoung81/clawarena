#!/usr/bin/env python3
"""
cr_driver.py - Clawed Roulette automated driver for ClawArena

Usage:
  python cr_driver.py --register --once
  python cr_driver.py --once
  python cr_driver.py --loop
  python cr_driver.py --register --once --verbose
  python cr_driver.py --config /path/to/config.json --loop
  python cr_driver.py --register --games 5
"""

import argparse
import json
import sys
import time
import datetime
import threading

import requests


def log(msg, verbose=False, always=False):
    if always or verbose:
        ts = datetime.datetime.now().strftime("%H:%M:%S")
        print(f"[{ts}] {msg}", flush=True)


def api_get(url, token, verbose=False):
    headers = {"Authorization": f"Bearer {token}"}
    resp = requests.get(url, headers=headers, timeout=15)
    resp.raise_for_status()
    data = resp.json()
    if verbose:
        log(f"GET {url} -> {resp.status_code}: {json.dumps(data)}", always=True)
    return data


def api_post(url, token, payload, verbose=False, expect_ok=True):
    headers = {
        "Authorization": f"Bearer {token}",
        "Content-Type": "application/json",
    }
    resp = requests.post(url, headers=headers, json=payload, timeout=15)
    data = resp.json() if resp.content else {}
    if verbose:
        log(f"POST {url} -> {resp.status_code}: {json.dumps(data)}", always=True)
    if expect_ok and resp.status_code >= 400:
        raise RuntimeError(f"POST {url} failed with {resp.status_code}: {data}")
    return data


def register_agent(auth_url, name, verbose=False):
    url = f"{auth_url}/auth/v1/agents/register"
    resp = requests.post(url, json={"name": name}, timeout=15)
    data = resp.json()
    if verbose:
        log(f"register {name} -> {resp.status_code}: {json.dumps(data)}", always=True)
    if resp.status_code not in (200, 201):
        raise RuntimeError(f"Registration failed for {name}: {data}")
    return data.get("agent_id") or data["id"], data["access_token"]


def choose_action(state, my_agent_id):
    """Pick the best action given current game state."""
    players = state.get("players", [])
    me = next((p for p in players if p["id"] == my_agent_id), None)
    if not me:
        return {"type": "fire", "target": 0}

    others_alive = [p for p in players if p["alive"] and p["seat"] != me["seat"]]
    turn_gadget_used = state.get("turn_gadget_used", False)

    # If a gadget was already used this turn, the follow-up shot is mandatory.
    if turn_gadget_used:
        if state.get("last_peek") == "blank":
            return {"type": "fire", "target": me["seat"]}
        target = others_alive[0]["seat"] if others_alive else me["seat"]
        return {"type": "fire", "target": target}

    # If we peeked and know next bullet
    if state.get("last_peek") == "live":
        target = others_alive[0]["seat"] if others_alive else me["seat"]
        return {"type": "fire", "target": target}
    if state.get("last_peek") == "blank":
        return {"type": "fire", "target": me["seat"]}  # extra turn

    # Use goggles first if available; it still leaves us one mandatory shot afterward.
    if "goggles" in (me.get("gadgets") or []):
        return {"type": "gadget", "gadget": "goggles"}

    # Heal if damaged; the turn continues with a forced shot afterward.
    if me["hits"] > 0 and "fish_chips" in (me.get("gadgets") or []):
        return {"type": "gadget", "gadget": "fish_chips"}

    # Default: fire at the most-damaged opponent
    if others_alive:
        target = max(others_alive, key=lambda p: p["hits"])["seat"]
        return {"type": "fire", "target": target}

    return {"type": "fire", "target": me["seat"]}


def setup_room(cfg, verbose=False):
    """Create room, join all agents, ready all. Updates cfg['room_id'] in place."""
    arena = cfg["arena_url"]
    agents = cfg["agents"]

    creator = agents[0]
    resp = api_post(
        f"{arena}/api/v1/rooms",
        creator["token"],
        {"game_type_id": cfg["game_type_id"]},
        verbose,
    )
    room_id = resp.get("id") or resp.get("room_id")
    if not room_id:
        raise RuntimeError(f"Room creation failed: {resp}")
    cfg["room_id"] = room_id
    log(f"Room {room_id} created", always=True)

    for agent in agents[1:]:
        api_post(f"{arena}/api/v1/rooms/{room_id}/join", agent["token"], {}, verbose)
        log(f"{agent['name']} joined", always=True)

    for agent in agents:
        api_post(f"{arena}/api/v1/rooms/{room_id}/ready", agent["token"], {}, verbose)
        log(f"{agent['name']} ready", always=True)

    # Resolve integer agent_ids assigned by the room
    room_data = api_get(f"{arena}/api/v1/rooms/{room_id}", creator["token"], verbose)
    if room_data.get("status") != "playing":
        raise RuntimeError(f"Room {room_id} did not start after all players readied: {room_data}")
    for room_agent in room_data.get("agents", []):
        for agent in agents:
            if room_agent["name"].startswith(agent["name"]):
                agent["agent_id"] = room_agent["agent_id"]
                log(f"Resolved {agent['name']} -> agent_id {agent['agent_id']}", verbose)


def sse_stream(url, headers):
    """Generator that yields {'event': ..., 'data': ...} dicts from an SSE endpoint."""
    resp = requests.get(url, headers=headers, stream=True, timeout=60)
    resp.raise_for_status()

    event_type = "message"
    data_lines = []

    for raw_line in resp.iter_lines(decode_unicode=True):
        if raw_line is None:
            continue

        line = raw_line if isinstance(raw_line, str) else raw_line.decode("utf-8")

        if line == "":
            if data_lines:
                yield {"event": event_type, "data": "\n".join(data_lines)}
            event_type = "message"
            data_lines = []
            continue

        if line.startswith(":"):
            continue

        if line.startswith("event:"):
            event_type = line[len("event:"):].strip()
        elif line.startswith("data:"):
            data_lines.append(line[len("data:"):].strip())


def ready_all(cfg, verbose=False):
    """POST /rooms/{room_id}/ready for every agent."""
    arena = cfg["arena_url"]
    room_id = cfg["room_id"]
    for agent in cfg["agents"]:
        api_post(f"{arena}/api/v1/rooms/{room_id}/ready", agent["token"], {}, verbose)
        log(f"{agent['name']} ready", always=True)


def play_game(cfg, verbose=False, slow=False):
    """SSE-based game loop. One thread per agent listens on the SSE stream."""
    arena = cfg["arena_url"]
    agents = cfg["agents"]
    room_id = cfg["room_id"]

    by_id = {a["agent_id"]: a for a in agents}
    game_done = threading.Event()

    def agent_sse_loop(agent):
        aid = agent["agent_id"]
        token = agent["token"]
        url = f"{arena}/api/v1/rooms/{room_id}/play"
        headers = {"Authorization": f"Bearer {token}", "Accept": "text/event-stream"}

        retries = 0
        max_retries = 10
        backoff = 1.0

        while not game_done.is_set() and retries <= max_retries:
            try:
                log(f"{agent['name']} connecting to SSE...", verbose)
                for event in sse_stream(url, headers):
                    if game_done.is_set():
                        return

                    evt_name = event["event"]

                    # Handle room lifecycle events
                    if evt_name == "room_event":
                        try:
                            data = json.loads(event["data"])
                        except (json.JSONDecodeError, TypeError):
                            continue
                        if data.get("game_over") or data.get("type") == "room_closed":
                            if not game_done.is_set():
                                log("Room closed", always=True)
                                game_done.set()
                            return
                        continue

                    if evt_name != "game_event":
                        continue

                    try:
                        data = json.loads(event["data"])
                    except (json.JSONDecodeError, TypeError):
                        continue

                    event_type = data.get("event_type", "")
                    seq = data.get("seq", "?")

                    if verbose:
                        log(f"{agent['name']} event seq={seq} type={event_type}", always=True)

                    # Check game over
                    if data.get("game_over"):
                        if not game_done.is_set():
                            state = data.get("state", {})
                            result = data.get("result", {}) or {}
                            winner_id = state.get("winner")
                            is_draw = state.get("is_draw", False)
                            if is_draw:
                                log("Result: DRAW", always=True)
                            elif winner_id is not None:
                                wname = by_id.get(winner_id, {}).get("name", f"agent {winner_id}")
                                log(f"Result: {wname} wins!", always=True)
                            elif result.get("winner_ids"):
                                wid = result["winner_ids"][0] if result["winner_ids"] else None
                                wname = by_id.get(wid, {}).get("name", f"agent {wid}")
                                log(f"Result: {wname} wins!", always=True)
                            else:
                                log("Game finished", always=True)
                            # Print final hit counts
                            for p in state.get("players", []):
                                pname = by_id.get(p["id"], {}).get("name", f"seat {p['seat']}")
                                log(f"  {pname}: {p['hits']} hits, alive={p['alive']}", always=True)
                            trophy_url = (result.get("trophy_url") or "")
                            if trophy_url and winner_id == aid:
                                download_trophy(trophy_url, agent["name"], "clawed_roulette")
                            game_done.set()
                        return

                    # Handle trophy event
                    if event_type == "trophy_awarded":
                        details = data.get("details", {})
                        trophy_url = details.get("trophy_url")
                        target = data.get("target", {})
                        target_id = target.get("agent_id")
                        if trophy_url and target_id == aid:
                            winner_name = details.get("winner_name", agent["name"])
                            download_trophy(trophy_url, winner_name, "clawed_roulette")
                        continue

                    # Check if it's my turn via pending_action
                    pa = data.get("pending_action")
                    if not pa or pa["player_id"] != aid:
                        continue

                    state = data.get("state", data)
                    action = choose_action(state, aid)
                    action_desc = format_action(action)
                    log(f"{agent['name']} -> {action_desc}", always=True)

                    if slow:
                        time.sleep(3)
                    result = api_post(
                        f"{arena}/api/v1/rooms/{room_id}/action",
                        token,
                        {"action": action},
                        verbose,
                        expect_ok=False,
                    )
                    if "error" in result:
                        log(f"Action error: {result}", always=True)

                    retries = 0
                    backoff = 1.0

                # Stream ended without game_over
                if not game_done.is_set():
                    log(f"{agent['name']} SSE stream ended, reconnecting...", verbose)
                    retries += 1
                    time.sleep(backoff)
                    backoff = min(backoff * 2, 30.0)

            except requests.exceptions.RequestException as e:
                if game_done.is_set():
                    return
                retries += 1
                log(f"{agent['name']} SSE error: {e}, retry {retries}/{max_retries}", always=True)
                time.sleep(backoff)
                backoff = min(backoff * 2, 30.0)

        if not game_done.is_set():
            log(f"{agent['name']} exhausted SSE retries", always=True)

    log(f"Game started in room {room_id} (SSE mode)", always=True)

    threads = []
    for agent in agents:
        t = threading.Thread(target=agent_sse_loop, args=(agent,), daemon=True)
        t.start()
        threads.append(t)

    game_done.wait(timeout=300)

    if not game_done.is_set():
        log("Timeout waiting for game to finish (SSE mode)", always=True)

    for t in threads:
        t.join(timeout=5)


def format_action(action):
    """Human-readable description of an action."""
    atype = action.get("type", "?")
    if atype == "fire":
        return f"fire at seat {action.get('target', '?')}"
    elif atype == "gadget":
        return f"use {action.get('gadget', '?')}"
    return str(action)


def do_register(cfg, verbose=False, creds_path=None):
    """Register agents with bare names and save credentials."""
    auth = cfg["auth_base_url"]
    for agent in cfg["agents"]:
        name = agent["name"]
        agent_id, token = register_agent(auth, name, verbose)
        agent["agent_id"] = agent_id
        agent["token"] = token
        log(f"Registered {name} -> {agent_id}", always=True)
    if creds_path:
        save_credentials(cfg["agents"], creds_path)


def load_config(path):
    with open(path) as f:
        return json.load(f)


def save_config(cfg, path):
    with open(path, "w") as f:
        json.dump(cfg, f, indent=2)


def save_credentials(agents, path):
    creds = {}
    for agent in agents:
        creds[agent["name"]] = {
            "agent_id": agent["agent_id"],
            "token": agent["token"],
        }
    with open(path, "w") as f:
        json.dump(creds, f, indent=2)
    log(f"Credentials saved to {path}", always=True)


def load_credentials(agents, path):
    try:
        with open(path) as f:
            creds = json.load(f)
    except FileNotFoundError:
        return False
    loaded = 0
    for agent in agents:
        if agent["name"] in creds:
            agent["agent_id"] = creds[agent["name"]]["agent_id"]
            agent["token"] = creds[agent["name"]]["token"]
            loaded += 1
    log(f"Loaded credentials for {loaded}/{len(agents)} agents from {path}", always=True)
    return loaded == len(agents)


def download_trophy(url, agent_name, game_type, output_dir="./trophies"):
    """Download the winner's trophy image."""
    try:
        import os
        os.makedirs(output_dir, exist_ok=True)
        resp = requests.get(url, timeout=15)
        resp.raise_for_status()
        ext = url.rsplit(".", 1)[-1] if "." in url else "svg"
        filename = f"{game_type}_{agent_name}.{ext}"
        filepath = os.path.join(output_dir, filename)
        with open(filepath, "wb") as f:
            f.write(resp.content)
        log(f"Trophy saved to {filepath}", always=True)
    except Exception as e:
        log(f"Failed to download trophy: {e}", always=True)


def main():
    parser = argparse.ArgumentParser(
        description="Clawed Roulette automated driver for ClawArena (SSE mode)",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
examples:
  python cr_driver.py --register --once
      Register fresh agents, play one game, exit.

  python cr_driver.py --once
      Use tokens from credentials.json, play one game, exit.

  python cr_driver.py --loop
      Play games forever (registers once, reuses credentials). Ctrl+C to stop.

  python cr_driver.py --register --once --verbose
      Register, play, and print all API responses.

  python cr_driver.py --config /path/to/config.json --once
      Use a custom config file.

  python cr_driver.py --register --games 5
      Register, play 5 games in the same room.

  python cr_driver.py --register --once --slow
      Register, play one game with 3s delay between actions (for live viewing).
        """,
    )
    parser.add_argument("--config", default="./config.json", help="Path to config.json (default: ./config.json)")
    parser.add_argument("--credentials", default="./credentials.json", help="Path to credentials.json (default: ./credentials.json)")
    parser.add_argument("--once", action="store_true", help="Play one game then exit (default if neither --once nor --loop)")
    parser.add_argument("--loop", action="store_true", help="Play games forever until Ctrl+C")
    parser.add_argument("--verbose", action="store_true", help="Print full API responses")
    parser.add_argument("--slow", action="store_true", help="Sleep 3s before each action (for human observation)")
    parser.add_argument("--register", action="store_true", help="Register new agents before playing")
    parser.add_argument("--games", type=int, default=1, help="Number of games to play in the same room (default: 1)")
    args = parser.parse_args()

    cfg = load_config(args.config)

    if args.loop:
        game_num = 0
        if not load_credentials(cfg["agents"], args.credentials):
            do_register(cfg, args.verbose, creds_path=args.credentials)
        try:
            while True:
                game_num += 1
                log(f"\n=== Game {game_num} ===", always=True)
                setup_room(cfg, args.verbose)
                play_game(cfg, args.verbose, slow=args.slow)
                time.sleep(2)
        except KeyboardInterrupt:
            log("\nStopped.", always=True)
    else:
        if args.register:
            do_register(cfg, args.verbose, creds_path=args.credentials)
        elif not load_credentials(cfg["agents"], args.credentials):
            if not cfg["agents"][0].get("token"):
                print("Error: no tokens found. Run with --register first, or provide credentials.json.", file=sys.stderr)
                sys.exit(1)
        setup_room(cfg, args.verbose)
        for game_num in range(1, args.games + 1):
            if args.games > 1:
                log(f"\n=== Game {game_num}/{args.games} ===", always=True)
            if game_num > 1:
                time.sleep(2)
                ready_all(cfg, args.verbose)
            play_game(cfg, args.verbose, slow=args.slow)


if __name__ == "__main__":
    main()
