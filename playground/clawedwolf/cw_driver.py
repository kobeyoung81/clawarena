#!/usr/bin/env python3
"""
cw_driver.py - ClawedWolf automated driver for ClawArena

Runs a single Python process that manages all 6 agents through every phase of
ClawedWolf (night/day cycles) until the game ends. Reads credentials from
config.json — no hardcoded tokens.

Usage:
  python cw_driver.py --register --once
  python cw_driver.py --once
  python cw_driver.py --loop
  python cw_driver.py --register --once --verbose
  python cw_driver.py --config /path/to/config.json --loop
  python cw_driver.py --register --games 3
"""

import argparse
import json
import random
import sys
import time
import datetime
import threading

import requests


SPEECH_TEMPLATES = {
    "clawedwolf": [
        "I don't have much to go on yet. Let's be careful with our vote.",
        "That's suspicious behavior. I think we should consider voting for them.",
        "I'm just a villager trying to figure things out like everyone else.",
    ],
    "seer": [
        "Based on what I know, I believe we should focus on the wolves among us.",
        "Trust me, I've been gathering intel. We need to act on it.",
        "I have some information to share with everyone.",
    ],
    "guard": [
        "I've been doing my best to keep everyone safe. Let's think about who's suspicious.",
        "I think we should focus on voting out the most suspicious player.",
        "Let's analyze who could be the wolves based on who's been killed.",
    ],
    "villager": [
        "I don't have any special information, but something feels off about some players.",
        "Let's think logically about who the wolves might be.",
        "I'm a villager and I trust the seer if they speak up.",
    ],
    "wolf_discuss": [
        "I think we should target someone less suspicious. Let's coordinate.",
        "Let's agree on the same target this time. How about we focus on the vocal ones?",
        "We need to be strategic. Let's pick someone the village won't defend.",
    ],
}


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


def api_post(url, token, payload, verbose=False):
    headers = {
        "Authorization": f"Bearer {token}",
        "Content-Type": "application/json",
    }
    resp = requests.post(url, headers=headers, json=payload, timeout=15)
    data = resp.json() if resp.content else {}
    if verbose:
        log(f"POST {url} -> {resp.status_code}: {json.dumps(data)}", always=True)
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


def setup_room(cfg, verbose=False):
    """Create room, all agents join, all agents ready. Updates cfg in place."""
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

    # Resolve integer agent_ids assigned by the room (registration returns string UUIDs)
    room_data = api_get(f"{arena}/api/v1/rooms/{room_id}", creator["token"], verbose)
    for room_agent in room_data.get("agents", []):
        for agent in agents:
            if room_agent["name"].startswith(agent["name"]):
                agent["agent_id"] = room_agent["agent_id"]
                log(f"Resolved {agent['name']} -> agent_id {agent['agent_id']}", verbose)


def get_wolf_seats(agents):
    return {a["seat"] for a in agents if a.get("role") == "clawedwolf" and a.get("seat") is not None}


def sse_stream(url, headers):
    """Generator that connects to an SSE endpoint and yields event dicts."""
    resp = requests.get(url, headers=headers, stream=True, timeout=60)
    resp.raise_for_status()

    event_type = ""
    data_lines = []

    for raw_line in resp.iter_lines(decode_unicode=True):
        if raw_line is None:
            continue

        line = raw_line if raw_line is not None else ""

        if line == "":
            # Empty line terminates an event
            if data_lines or event_type:
                yield {
                    "event": event_type or "message",
                    "data": "\n".join(data_lines),
                }
                event_type = ""
                data_lines = []
            continue

        if line.startswith(":"):
            # Comment / keepalive — ignore
            continue

        if line.startswith("event:"):
            event_type = line[len("event:"):].strip()
        elif line.startswith("data:"):
            data_lines.append(line[len("data:"):].strip())

    # Yield any remaining event when stream ends
    if data_lines or event_type:
        yield {
            "event": event_type or "message",
            "data": "\n".join(data_lines),
        }


def play_game(cfg, verbose=False, slow=False):
    """SSE-based game loop for ClawedWolf. One thread per agent."""
    arena = cfg["arena_url"]
    agents = cfg["agents"]
    room_id = cfg["room_id"]

    # Shared game-scoped state (safe: server enforces turn order)
    shared = {
        "seer_investigated": set(),
        "guard_last_protected": None,
        "speech_idx": {},
        "seer_results_cache": {},
        "round_num": 0,
    }

    game_done = threading.Event()

    log(f"Game started (SSE) in room {room_id}", always=True)

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

                    evt_name = event["event"]  # "game_event", "room_event", or "message"

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

                    # Only process game events
                    if evt_name != "game_event":
                        continue  # Skip keepalives and unknown events

                    try:
                        data = json.loads(event["data"])
                    except (json.JSONDecodeError, TypeError):
                        continue

                    event_type = data.get("event_type", "")
                    seq = data.get("seq", "?")

                    if verbose:
                        log(f"{agent['name']} event seq={seq} type={event_type}", always=True)

                    # --- Role discovery ---
                    # Method 1: From roles_assigned event during catch-up
                    if event_type == "roles_assigned" and data.get("details", {}).get("role"):
                        agent["role"] = data["details"]["role"]
                        log(f"{agent['name']} assigned role: {agent['role']}", verbose)

                    # Method 2: From player view state (more reliable)
                    state = data.get("state", {})
                    if state.get("your_role") and not agent.get("role"):
                        agent["role"] = state["your_role"]
                        agent["seat"] = state.get("your_seat")
                        log(f"{agent['name']} discovered role: {agent['role']} seat: {agent.get('seat')}", verbose)

                    # Update seat from state if not yet set
                    if state.get("your_seat") is not None and agent.get("seat") is None:
                        agent["seat"] = state["your_seat"]

                    # Update alive status from players array in state
                    for player in state.get("players", []):
                        for a in agents:
                            if a.get("seat") == player.get("seat"):
                                a["alive"] = player.get("alive", True)

                    # Check game over
                    if data.get("game_over"):
                        if not game_done.is_set():
                            result = data.get("result", {}) or {}
                            winner = (
                                state.get("winner")
                                or result.get("winner_team")
                                or "unknown"
                            )
                            log(f"\nGAME OVER! Winner: {winner}", always=True)
                            # Print any details from the state
                            details = data.get("details", {})
                            if details.get("message"):
                                log(f"  {details['message']}", always=True)
                            # Download trophy if URL provided
                            trophy_url = result.get("trophy_url")
                            if trophy_url:
                                download_trophy(trophy_url, name, "clawedwolf")
                            game_done.set()
                        return

                    # Handle trophy event
                    if event_type == "trophy_awarded":
                        details = data.get("details", {})
                        trophy_url = details.get("trophy_url")
                        target = data.get("target", {})
                        target_id = target.get("agent_id")
                        if trophy_url and target_id == aid:
                            winner_name = details.get("winner_name", name)
                            download_trophy(trophy_url, winner_name, "clawedwolf")
                        continue

                    # Check if it's my turn via pending_action
                    pa = data.get("pending_action")
                    if not pa or pa["player_id"] != aid:
                        retries = 0
                        backoff = 1.0
                        continue

                    phase = state.get("phase", "")
                    cur_round = state.get("round", 0)
                    role = agent.get("role", "villager")
                    name = agent["name"]
                    valid = pa.get("valid_targets", [])

                    if cur_round != shared["round_num"]:
                        shared["round_num"] = cur_round
                        log(f"\n{'='*50}", always=True)
                        log(f"Round {cur_round}", always=True)
                        log(f"{'='*50}", always=True)

                    action = None

                    if phase == "night_clawedwolf":
                        wolf_seats = get_wolf_seats(agents)
                        targets = [s for s in valid if s not in wolf_seats and s != -1]
                        if not targets:
                            targets = [s for s in valid if s != -1] or valid
                        target = targets[0] if targets else valid[0]
                        action = {"action": {"type": "kill_vote", "target_seat": target}}
                        log(f"  Wolf: {name} votes to kill seat {target}", always=True)

                    elif phase == "night_wolf_discuss":
                        idx = shared["speech_idx"].get(f"wolf_{aid}", 0)
                        templates = SPEECH_TEMPLATES.get("wolf_discuss", SPEECH_TEMPLATES["clawedwolf"])
                        msg = templates[idx % len(templates)]
                        shared["speech_idx"][f"wolf_{aid}"] = idx + 1
                        action = {"action": {"type": "wolf_speak", "message": msg}}
                        short = msg[:60] + "..." if len(msg) > 60 else msg
                        log(f"  Wolf discuss: {name} says: \"{short}\"", always=True)

                    elif phase == "night_seer":
                        targets = [s for s in valid if s not in shared["seer_investigated"]]
                        if not targets:
                            targets = list(valid)
                        target = targets[0] if targets else valid[0]
                        shared["seer_investigated"].add(target)
                        action = {"action": {"type": "investigate", "target_seat": target}}
                        log(f"  Seer: {name} investigates seat {target}", always=True)

                    elif phase == "night_guard":
                        targets = [s for s in valid if s != shared["guard_last_protected"]]
                        if not targets:
                            targets = list(valid)
                        target = targets[0] if targets else valid[0]
                        shared["guard_last_protected"] = target
                        action = {"action": {"type": "protect", "target_seat": target}}
                        log(f"  Guard: {name} protects seat {target}", always=True)

                    elif phase == "day_discuss":
                        idx = shared["speech_idx"].get(aid, 0)

                        if role == "seer":
                            sr = state.get("seer_results", {})
                            shared["seer_results_cache"].update(sr)
                            if shared["seer_results_cache"]:
                                findings = ", ".join(
                                    [f"seat {k} is {v}" for k, v in shared["seer_results_cache"].items()]
                                )
                                msg = f"I am the Seer. My findings: {findings}. Let us vote wisely."
                            else:
                                templates = SPEECH_TEMPLATES.get("seer", SPEECH_TEMPLATES["villager"])
                                msg = templates[idx % len(templates)]
                        else:
                            templates = SPEECH_TEMPLATES.get(role, SPEECH_TEMPLATES["villager"])
                            msg = templates[idx % len(templates)]

                        shared["speech_idx"][aid] = idx + 1
                        action = {"action": {"type": "speak", "message": msg}}
                        short = msg[:60] + "..." if len(msg) > 60 else msg
                        log(f"  {name} speaks: \"{short}\"", always=True)

                    elif phase == "day_vote":
                        if role == "clawedwolf":
                            wolf_seats = get_wolf_seats(agents)
                            targets = [s for s in valid if s not in wolf_seats and s != -1]
                        else:
                            targets = [s for s in valid if s != agent.get("seat") and s != -1]
                            if role == "seer" and shared["seer_results_cache"]:
                                evil_seats = [
                                    int(k) for k, v in shared["seer_results_cache"].items() if v == "evil"
                                ]
                                evil_targets = [s for s in evil_seats if s in targets]
                                if evil_targets:
                                    targets = evil_targets

                        target = targets[0] if targets else (valid[0] if valid else -1)
                        action = {"action": {"type": "vote", "target_seat": target}}
                        log(f"  {name} votes for seat {target}", always=True)

                    else:
                        log(f"  Waiting: {name} in phase {phase}", always=True)
                        retries = 0
                        backoff = 1.0
                        continue

                    if action:
                        if slow:
                            time.sleep(3)
                        result = api_post(
                            f"{arena}/api/v1/rooms/{room_id}/action", token, action, verbose
                        )
                        if "error" in result:
                            log(f"  Action error: {result}", always=True)
                        if result.get("game_over"):
                            log(f"\nGAME OVER after action!", always=True)
                            r = result.get("result", {}) or {}
                            winner = r.get("winner_team", "unknown")
                            log(f"Winner: {winner}", always=True)
                            game_done.set()
                            return

                    retries = 0
                    backoff = 1.0

                # Stream ended normally
                if not game_done.is_set():
                    retries += 1
                    log(f"{agent['name']} SSE stream ended, retry {retries}/{max_retries}", verbose)
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

    threads = []
    for agent in agents:
        t = threading.Thread(target=agent_sse_loop, args=(agent,), daemon=True)
        t.start()
        threads.append(t)

    game_done.wait(timeout=600)

    if not game_done.is_set():
        log("SSE game timeout (600s) -- game may be stuck", always=True)

    for t in threads:
        t.join(timeout=5)


def ready_all(cfg, verbose=False):
    """POST /rooms/{room_id}/ready for all agents."""
    arena = cfg["arena_url"]
    room_id = cfg["room_id"]
    for agent in cfg["agents"]:
        api_post(f"{arena}/api/v1/rooms/{room_id}/ready", agent["token"], {}, verbose)
        log(f"{agent['name']} ready", always=True)


def do_register(cfg, verbose=False, creds_path=None):
    """Register agents with bare names and save credentials."""
    auth = cfg["auth_base_url"]
    for agent in cfg["agents"]:
        name = agent["name"]
        agent_id, token = register_agent(auth, name, verbose)
        agent["agent_id"] = agent_id
        agent["token"] = token
        agent["seat"] = None
        agent["role"] = ""
        agent["alive"] = True
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
        description="ClawedWolf automated driver for ClawArena (SSE mode)",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
examples:
  python cw_driver.py --register --once
      Register fresh agents, play one game, exit.

  python cw_driver.py --once
      Use tokens from credentials.json, play one game, exit.

  python cw_driver.py --loop
      Play games forever (registers once, reuses credentials). Ctrl+C to stop.

  python cw_driver.py --register --once --verbose
      Register, play, and print all API responses.

  python cw_driver.py --config /path/to/config.json --once
      Use a custom config file.

  python cw_driver.py --register --games 3
      Register, play 3 games in the same room.

  python cw_driver.py --register --once --slow
      Register, play one game with 3s delay between moves (for live viewing).
        """,
    )
    parser.add_argument("--config", default="./config.json", help="Path to config.json (default: ./config.json)")
    parser.add_argument("--credentials", default="./credentials.json", help="Path to credentials.json (default: ./credentials.json)")
    parser.add_argument("--once", action="store_true", help="Play one game then exit (default if neither flag set)")
    parser.add_argument("--loop", action="store_true", help="Play games forever until Ctrl+C")
    parser.add_argument("--verbose", action="store_true", help="Print full API responses")
    parser.add_argument("--slow", action="store_true", help="Sleep 3s before each move (for human observation)")
    parser.add_argument("--register", action="store_true", help="Register new agents before playing")
    parser.add_argument("--games", type=int, default=1, help="Number of games to play in the same room (default: 1)")
    args = parser.parse_args()

    cfg = load_config(args.config)

    if args.loop:
        game_num = 0
        # Register once (or load existing credentials), then reuse for all games
        if not load_credentials(cfg["agents"], args.credentials):
            do_register(cfg, args.verbose, creds_path=args.credentials)
        try:
            while True:
                game_num += 1
                log(f"\n=== Game {game_num} ===", always=True)
                for agent in cfg["agents"]:
                    agent["seat"] = None
                    agent["role"] = ""
                    agent["alive"] = True
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
        # Reset alive/seat/role for fresh game
        for agent in cfg["agents"]:
            agent.setdefault("alive", True)
        setup_room(cfg, args.verbose)
        play_game(cfg, args.verbose, slow=args.slow)

        for game_i in range(2, args.games + 1):
            log(f"\n=== Game {game_i}/{args.games} (same room) ===", always=True)
            for agent in cfg["agents"]:
                agent["seat"] = None
                agent["role"] = ""
                agent["alive"] = True
            time.sleep(2)
            ready_all(cfg, args.verbose)
            play_game(cfg, args.verbose, slow=args.slow)


if __name__ == "__main__":
    main()
