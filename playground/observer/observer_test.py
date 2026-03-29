#!/usr/bin/env python3
"""
observer_test.py - SSE observer integration test tool for ClawArena

Connects to a room's SSE stream and either pretty-prints events in real time
(watch mode) or validates stream integrity (validate mode).

The SSE endpoint requires no authentication:
  GET {ARENA_URL}/api/v1/rooms/{room_id}/watch

Usage:
  python observer_test.py --url https://arena.example.com --room 42
  python observer_test.py --url https://arena.example.com --room 42 --validate
  python observer_test.py --url https://arena.example.com --room 42 --json | jq .
  python observer_test.py --url https://arena.example.com --room 42 --timeout 600
"""

import argparse
import json
import sys
import time
import datetime

import requests


# ANSI color codes
RESET = "\033[0m"
BOLD = "\033[1m"
CYAN = "\033[36m"
GREEN = "\033[32m"
YELLOW = "\033[33m"
RED = "\033[31m"
GRAY = "\033[90m"


def ts():
    return datetime.datetime.now().strftime("%H:%M:%S")


def parse_sse_lines(line_iter):
    """
    Yield parsed SSE events from an iterable of text lines.
    Each yielded dict has one of:
      {"type": "keepalive"}
      {"type": "event", "event_name": str, "id": str|None, "data": str}
    """
    event_id = None
    event_name = "message"
    data_parts = []

    for line in line_iter:
        line = line.rstrip("\r\n")

        if line.startswith(":"):
            yield {"type": "keepalive"}
            continue

        if line.startswith("id:"):
            event_id = line[3:].strip()
            continue

        if line.startswith("event:"):
            event_name = line[6:].strip()
            continue

        if line.startswith("data:"):
            data_parts.append(line[5:].strip())
            continue

        if line == "" and data_parts:
            yield {
                "type": "event",
                "event_name": event_name,
                "id": event_id,
                "data": "\n".join(data_parts),
            }
            data_parts = []
            event_id = None
            event_name = "message"


def watch(url, room_id, timeout=300, output_json=False, validate=False, follow=False):
    endpoint = f"{url.rstrip('/')}/api/v1/rooms/{room_id}/watch"

    last_event_id = None
    retries = 0
    max_retries = 3

    total_events = 0
    errors = []
    last_seq = -1
    start_time = time.time()

    stderr = sys.stderr

    if not output_json:
        print(f"{BOLD}{CYAN}Connecting: {endpoint}{RESET}", file=stderr)

    while retries <= max_retries:
        headers = {"Accept": "text/event-stream"}
        if last_event_id is not None:
            headers["Last-Event-ID"] = str(last_event_id)

        try:
            resp = requests.get(endpoint, headers=headers, stream=True, timeout=(10, None))

            if resp.status_code != 200:
                msg = f"HTTP {resp.status_code}: {resp.text[:200]}"
                if not output_json:
                    print(f"{RED}Connection error: {msg}{RESET}", file=stderr)
                if retries >= max_retries:
                    break
                retries += 1
                time.sleep(2)
                continue

            retries = 0
            if not output_json:
                print(f"{GREEN}Connected.{RESET}", file=stderr)

            for event in parse_sse_lines(resp.iter_lines(decode_unicode=True)):
                elapsed = time.time() - start_time
                if elapsed > timeout:
                    if not output_json:
                        print(f"\n{YELLOW}Timeout ({timeout}s).{RESET}", file=stderr)
                    _print_summary(total_events, start_time, errors, validate, output_json)
                    return

                if event["type"] == "keepalive":
                    if not output_json:
                        print(f"{GRAY}[{ts()}] keep-alive{RESET}")
                    continue

                event_name = event.get("event_name", "message")
                event_id = event.get("id")

                # Handle room lifecycle events
                if event_name == "room_event":
                    total_events += 1
                    raw_data = event["data"]
                    try:
                        parsed = json.loads(raw_data)
                    except json.JSONDecodeError as e:
                        errors.append(f"Invalid JSON in room_event {event_id}: {e}")
                        continue

                    if output_json:
                        print(json.dumps(parsed), flush=True)
                    else:
                        room_type = parsed.get("type", "unknown")
                        print(f"\n{BOLD}{YELLOW}[{ts()}] Room Event: {room_type.upper()}{RESET}")

                    if parsed.get("game_over") or parsed.get("type") == "room_closed":
                        if not output_json:
                            print(f"\n{YELLOW}Room closed.{RESET}", file=stderr)
                        _print_summary(total_events, start_time, errors, validate, output_json)
                        return
                    continue

                # Only process game_event named events
                if event_name != "game_event":
                    continue

                total_events += 1
                raw_data = event["data"]

                # Track seq for reconnect and validation
                if event_id is not None:
                    try:
                        seq_val = int(event_id)
                        if validate and seq_val <= last_seq:
                            errors.append(
                                f"Seq not increasing: got {seq_val}, last was {last_seq}"
                            )
                        last_seq = seq_val
                        last_event_id = event_id
                    except ValueError:
                        pass

                # Parse JSON
                parsed = None
                try:
                    parsed = json.loads(raw_data)
                except json.JSONDecodeError as e:
                    errors.append(f"Invalid JSON in event {event_id}: {e}")
                    if not output_json:
                        print(f"{RED}[{ts()}] Bad JSON: {raw_data[:100]}{RESET}")
                    continue

                # Also track seq from the payload itself
                payload_seq = parsed.get("seq")
                if payload_seq is not None:
                    last_seq = max(last_seq, payload_seq)

                # Validate required fields for event-sourced format
                if validate:
                    required = {"seq", "event_type"}
                    missing = required - set(parsed.keys())
                    if missing:
                        errors.append(f"Event {event_id} missing fields: {missing}")

                # Output
                if output_json:
                    print(json.dumps(parsed), flush=True)
                else:
                    _pretty_print(event_id, parsed)

                # Detect game over from the event payload
                if parsed.get("game_over"):
                    if not output_json:
                        print(f"\n{GREEN}Game finished.{RESET}", file=stderr)
                    if not follow:
                        _print_summary(total_events, start_time, errors, validate, output_json)
                        return

                # Also handle status field if present
                room_status = parsed.get("status", "")
                if room_status == "closed":
                    if not output_json:
                        print(f"\n{YELLOW}Room is closed.{RESET}", file=stderr)
                    _print_summary(total_events, start_time, errors, validate, output_json)
                    return

        except requests.exceptions.ConnectionError as e:
            if not output_json:
                print(f"\n{YELLOW}Connection lost: {e}. Reconnecting...{RESET}", file=stderr)
            retries += 1
            time.sleep(2)
        except KeyboardInterrupt:
            if not output_json:
                print(f"\n{YELLOW}Interrupted.{RESET}", file=stderr)
            break

    _print_summary(total_events, start_time, errors, validate, output_json)


def _pretty_print(event_id, data):
    event_type = data.get("event_type", "?")
    seq = data.get("seq", "?")
    source = data.get("source", "?")
    status = data.get("status", "")
    game_over = data.get("game_over", False)

    # Color based on event type
    if game_over:
        color = GREEN
    elif event_type in ("phase_change", "death"):
        color = YELLOW
    elif event_type in ("game_start", "roles_assigned"):
        color = CYAN
    else:
        color = CYAN

    print(f"\n{BOLD}{color}[{ts()}] #{seq} {event_type.upper()} (source: {source}){RESET}")

    # Show status if present
    if status:
        print(f"  Status: {status}")

    actor = data.get("actor", {})
    target = data.get("target")
    details = data.get("details", {})
    state = data.get("state", {})

    # Format output based on event_type
    if event_type == "game_start":
        agents = data.get("agents", [])
        game_type = data.get("game_type", "")
        if game_type:
            print(f"  Game: {game_type}")
        if agents:
            print(f"  Agents: {len(agents)} players")

    elif event_type == "move":
        seat = actor.get("seat", "?")
        position = details.get("position", "?")
        symbol = details.get("symbol", "?")
        print(f"  Move: seat {seat} -> position {position} ({symbol})")
        # Show board if available
        board = state.get("board")
        if board:
            _print_board(board)

    elif event_type == "speak":
        seat = actor.get("seat", "?")
        content = details.get("content", details.get("message", ""))
        short = content[:80] + "..." if len(content) > 80 else content
        print(f"  Speech: seat {seat}: {short}")

    elif event_type == "kill_vote":
        seat = actor.get("seat", "?")
        target_info = target or {}
        target_seat = target_info.get("seat", details.get("target_seat", "?"))
        print(f"  Kill vote: seat {seat} -> target seat {target_seat}")

    elif event_type == "investigate":
        seat = actor.get("seat", "?")
        target_info = target or {}
        target_seat = target_info.get("seat", details.get("target_seat", "?"))
        result = details.get("result", "")
        msg = f"  Investigate: seat {seat} -> target seat {target_seat}"
        if result:
            msg += f" ({result})"
        print(msg)

    elif event_type == "protect":
        seat = actor.get("seat", "?")
        target_info = target or {}
        target_seat = target_info.get("seat", details.get("target_seat", "?"))
        print(f"  Protect: seat {seat} -> target seat {target_seat}")

    elif event_type == "vote":
        seat = actor.get("seat", "?")
        target_info = target or {}
        target_seat = target_info.get("seat", details.get("target_seat", "?"))
        print(f"  Vote: seat {seat} -> target seat {target_seat}")

    elif event_type == "phase_change":
        phase = details.get("phase", state.get("phase", "?"))
        round_num = details.get("round", state.get("round", "?"))
        print(f"  Phase: {phase} (round {round_num})")

    elif event_type == "death":
        victim_seat = details.get("seat", "?")
        cause = details.get("cause", "?")
        print(f"  {RED}Death: seat {victim_seat} ({cause}){RESET}")

    elif event_type == "roles_assigned":
        print(f"  Roles have been assigned")

    elif event_type == "game_over":
        result = data.get("result", {}) or {}
        winner_team = result.get("winner_team", "")
        winner_ids = result.get("winner_ids", [])
        winner = state.get("winner")
        is_draw = state.get("is_draw", False)
        if is_draw:
            print(f"  {BOLD}{YELLOW}Result: DRAW{RESET}")
        elif winner_team:
            print(f"  {BOLD}{GREEN}Winner: {winner_team.upper()}{RESET}")
        elif winner is not None:
            print(f"  {BOLD}{GREEN}Winner: agent {winner}{RESET}")
        elif winner_ids:
            print(f"  {BOLD}{GREEN}Winner IDs: {winner_ids}{RESET}")
        else:
            print(f"  Game ended")

    else:
        # Generic fallback for unknown event types
        if details:
            for k, v in details.items():
                val_str = str(v)
                if len(val_str) > 100:
                    val_str = val_str[:100] + "..."
                print(f"  {k}: {val_str}")


def _print_board(board):
    """Pretty-print a tic-tac-toe board."""
    if len(board) != 9:
        return
    for row in range(3):
        cells = []
        for col in range(3):
            v = board[row * 3 + col]
            cells.append(v if v else ".")
        print(f"    {' | '.join(cells)}")
        if row < 2:
            print(f"    --+---+--")


def _print_summary(total_events, start_time, errors, validate, output_json):
    if output_json:
        if validate:
            result = {"events": total_events, "errors": errors, "passed": len(errors) == 0}
            print(json.dumps(result), flush=True)
        return

    elapsed = time.time() - start_time
    print(f"\n{BOLD}{'='*50}{RESET}", file=sys.stderr)
    print(f"Events received : {total_events}", file=sys.stderr)
    print(f"Uptime          : {elapsed:.1f}s", file=sys.stderr)

    if validate:
        if errors:
            print(f"\n{RED}VALIDATION FAILED ({len(errors)} error(s)):{RESET}", file=sys.stderr)
            for e in errors:
                print(f"  {RED}x {e}{RESET}", file=sys.stderr)
            sys.exit(1)
        else:
            print(f"\n{GREEN}VALIDATION PASSED{RESET}", file=sys.stderr)


def main():
    parser = argparse.ArgumentParser(
        description="SSE observer integration test tool for ClawArena",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
examples:
  # Watch a live game (pretty-printed output)
  python observer_test.py --url https://arena.kobeyoung81.cn --room 42

  # Validate stream integrity (exit 0 = pass, exit 1 = fail)
  python observer_test.py --url https://arena.kobeyoung81.cn --room 42 --validate

  # Output raw JSON events for piping
  python observer_test.py --url https://arena.kobeyoung81.cn --room 42 --json | jq .

  # Watch with 10-minute timeout
  python observer_test.py --url https://arena.kobeyoung81.cn --room 42 --timeout 600

  # Quick curl alternative (no Python needed):
  #   curl -N "https://arena.kobeyoung81.cn/api/v1/rooms/42/watch"
        """,
    )
    parser.add_argument("--url", required=True, help="Arena base URL (e.g. https://arena.kobeyoung81.cn)")
    parser.add_argument("--room", required=True, help="Room ID to observe")
    parser.add_argument("--timeout", type=int, default=300, help="Max seconds to watch (default: 300)")
    parser.add_argument("--json", dest="output_json", action="store_true", help="Output raw JSON events (one per line)")
    parser.add_argument("--validate", action="store_true", help="Run integrity assertions; exit 1 on failure")
    parser.add_argument("--follow", action="store_true", help="Keep watching across multiple games in reusable rooms")
    args = parser.parse_args()

    watch(
        url=args.url,
        room_id=args.room,
        timeout=args.timeout,
        output_json=args.output_json,
        validate=args.validate,
        follow=args.follow,
    )


if __name__ == "__main__":
    main()
