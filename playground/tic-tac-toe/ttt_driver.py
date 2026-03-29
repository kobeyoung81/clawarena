#!/usr/bin/env python3
"""
ttt_driver.py - Tic-Tac-Toe automated driver for ClawArena

Usage:
  python ttt_driver.py --register --once
  python ttt_driver.py --once
  python ttt_driver.py --loop
  python ttt_driver.py --register --once --verbose
  python ttt_driver.py --config /path/to/config.json --loop
  python ttt_driver.py --register --games 5
"""

import argparse
import json
import sys
import time
import datetime
import threading

import requests


WIN_LINES = [
    (0, 1, 2), (3, 4, 5), (6, 7, 8),
    (0, 3, 6), (1, 4, 7), (2, 5, 8),
    (0, 4, 8), (2, 4, 6),
]


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


def choose_move(board, valid_targets, my_mark, opp_mark):
    """Win -> block -> center -> corner -> edge."""
    # Win
    for line in WIN_LINES:
        empties = [p for p in line if board[p] == ""]
        my_cells = [p for p in line if board[p] == my_mark]
        if len(my_cells) == 2 and len(empties) == 1 and empties[0] in valid_targets:
            return empties[0]
    # Block
    for line in WIN_LINES:
        empties = [p for p in line if board[p] == ""]
        opp_cells = [p for p in line if board[p] == opp_mark]
        if len(opp_cells) == 2 and len(empties) == 1 and empties[0] in valid_targets:
            return empties[0]
    # Center
    if 4 in valid_targets:
        return 4
    # Corners
    for c in [0, 2, 6, 8]:
        if c in valid_targets:
            return c
    # Edge fallback
    return valid_targets[0]


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

    # Resolve integer agent_ids assigned by the room (registration returns string UUIDs)
    room_data = api_get(f"{arena}/api/v1/rooms/{room_id}", creator["token"], verbose)
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
            # Empty line terminates the event
            if data_lines:
                yield {"event": event_type, "data": "\n".join(data_lines)}
            event_type = "message"
            data_lines = []
            continue

        if line.startswith(":"):
            # Comment / keepalive
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
    marks = {a["agent_id"]: a["mark"] for a in agents}

    game_done = threading.Event()

    def agent_sse_loop(agent):
        aid = agent["agent_id"]
        token = agent["token"]
        my_mark = marks[aid]
        opp_mark = next((m for k, m in marks.items() if k != aid), "O")
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

                    evt_type = event["event"]

                    if evt_type == "game_start":
                        log(f"{agent['name']} received game_start", verbose)
                        continue

                    if evt_type == "game_over":
                        try:
                            data = json.loads(event["data"])
                        except (json.JSONDecodeError, TypeError):
                            data = {}
                        if not game_done.is_set():
                            gs = data.get("state", data)
                            winner_id = gs.get("winner")
                            is_draw = gs.get("is_draw", False)
                            if is_draw:
                                log("Result: DRAW", always=True)
                            elif winner_id is not None:
                                wname = by_id.get(winner_id, {}).get("name", f"agent {winner_id}")
                                log(f"Result: {wname} ({marks.get(winner_id, '?')}) wins!", always=True)
                            else:
                                log("Game finished", always=True)
                            game_done.set()
                        return

                    if evt_type == "state":
                        try:
                            data = json.loads(event["data"])
                        except (json.JSONDecodeError, TypeError):
                            continue

                        if data.get("status") in ("finished", "closed", "intermission"):
                            if not game_done.is_set():
                                gs = data.get("state", {})
                                winner_id = gs.get("winner")
                                is_draw = gs.get("is_draw", False)
                                if is_draw:
                                    log("Result: DRAW", always=True)
                                elif winner_id is not None:
                                    wname = by_id.get(winner_id, {}).get("name", f"agent {winner_id}")
                                    log(f"Result: {wname} ({marks.get(winner_id, '?')}) wins!", always=True)
                                else:
                                    log("Game finished", always=True)
                                game_done.set()
                            return

                        pa = data.get("pending_action")
                        if not pa or pa["player_id"] != aid:
                            continue

                        board = data["state"]["board"]
                        valid = pa.get("valid_targets") or [i for i, v in enumerate(board) if v == ""]
                        position = choose_move(board, valid, my_mark, opp_mark)
                        log(f"{agent['name']} ({my_mark}) -> {position}", always=True)

                        if slow:
                            time.sleep(3)
                        result = api_post(
                            f"{arena}/api/v1/rooms/{room_id}/action",
                            token,
                            {"action": {"position": position}},
                            verbose,
                        )
                        if "error" in result:
                            log(f"Action error: {result}", always=True)

                    # Reset backoff on successful event processing
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


def do_register(cfg, verbose=False, suffix=None):
    """Register agents and update cfg in place."""
    auth = cfg["auth_base_url"]
    ts = suffix or str(int(time.time()))
    for agent in cfg["agents"]:
        name = f"{agent['name']}-{ts}"
        agent_id, token = register_agent(auth, name, verbose)
        agent["agent_id"] = agent_id
        agent["token"] = token
        log(f"Registered {name} -> {agent_id}", always=True)


def load_config(path):
    with open(path) as f:
        return json.load(f)


def save_config(cfg, path):
    with open(path, "w") as f:
        json.dump(cfg, f, indent=2)


def main():
    parser = argparse.ArgumentParser(
        description="Tic-Tac-Toe automated driver for ClawArena (SSE mode)",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
examples:
  python ttt_driver.py --register --once
      Register fresh agents, play one game, exit.

  python ttt_driver.py --once
      Use tokens already in config.json, play one game, exit.

  python ttt_driver.py --loop
      Play games forever (re-registers each round). Ctrl+C to stop.

  python ttt_driver.py --register --once --verbose
      Register, play, and print all API responses.

  python ttt_driver.py --config /path/to/config.json --once
      Use a custom config file.

  python ttt_driver.py --register --games 5
      Register, play 5 games in the same room.

  python ttt_driver.py --register --once --slow
      Register, play one game with 3s delay between moves (for live viewing).
        """,
    )
    parser.add_argument("--config", default="./config.json", help="Path to config.json (default: ./config.json)")
    parser.add_argument("--once", action="store_true", help="Play one game then exit (default if neither --once nor --loop)")
    parser.add_argument("--loop", action="store_true", help="Play games forever until Ctrl+C")
    parser.add_argument("--verbose", action="store_true", help="Print full API responses")
    parser.add_argument("--slow", action="store_true", help="Sleep 3s before each move (for human observation)")
    parser.add_argument("--register", action="store_true", help="Register new agents before playing")
    parser.add_argument("--games", type=int, default=1, help="Number of games to play in the same room (default: 1)")
    args = parser.parse_args()

    cfg = load_config(args.config)

    if args.loop:
        game_num = 0
        try:
            while True:
                game_num += 1
                log(f"\n=== Game {game_num} ===", always=True)
                do_register(cfg, args.verbose)
                save_config(cfg, args.config)
                setup_room(cfg, args.verbose)
                play_game(cfg, args.verbose, slow=args.slow)
                time.sleep(2)
        except KeyboardInterrupt:
            log("\nStopped.", always=True)
    else:
        if args.register:
            do_register(cfg, args.verbose)
            save_config(cfg, args.config)
        if not cfg["agents"][0].get("token"):
            print("Error: no tokens in config. Run with --register first.", file=sys.stderr)
            sys.exit(1)
        setup_room(cfg, args.verbose)
        for game_num in range(1, args.games + 1):
            if args.games > 1:
                log(f"\n=== Game {game_num}/{args.games} ===", always=True)
            if game_num > 1:
                time.sleep(2)  # Allow room state to settle
                ready_all(cfg, args.verbose)
            play_game(cfg, args.verbose, slow=args.slow)


if __name__ == "__main__":
    main()
