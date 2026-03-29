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
      {"type": "event", "id": str|None, "data": str}
    """
    event_id = None
    data_parts = []

    for line in line_iter:
        line = line.rstrip("\r\n")

        if line.startswith(":"):
            yield {"type": "keepalive"}
            continue

        if line.startswith("id:"):
            event_id = line[3:].strip()
            continue

        if line.startswith("data:"):
            data_parts.append(line[5:].strip())
            continue

        if line == "" and data_parts:
            yield {"type": "event", "id": event_id, "data": "\n".join(data_parts)}
            data_parts = []
            event_id = None


def watch(url, room_id, timeout=300, output_json=False, validate=False, follow=False):
    endpoint = f"{url.rstrip('/')}/api/v1/rooms/{room_id}/watch"

    last_event_id = None
    retries = 0
    max_retries = 3

    total_events = 0
    errors = []
    last_turn = -1
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

                total_events += 1
                raw_data = event["data"]
                event_id = event.get("id")

                # Track turn ID for reconnect and validation
                if event_id is not None:
                    try:
                        turn = int(event_id)
                        if validate and turn <= last_turn:
                            errors.append(
                                f"Turn ID not increasing: got {turn}, last was {last_turn}"
                            )
                        last_turn = turn
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

                # Validate required fields: every event must have room_id, turn, status
                if validate:
                    required = {"room_id", "turn", "status"}
                    missing = required - set(parsed.keys())
                    if missing:
                        errors.append(f"Event {event_id} missing fields: {missing}")

                # Output
                if output_json:
                    print(json.dumps(parsed), flush=True)
                else:
                    _pretty_print(event_id, parsed)

                # Handle room lifecycle events
                room_status = parsed.get("status", "")
                if room_status == "intermission" or parsed.get("game_over"):
                    if not output_json:
                        print(f"\n{GREEN}Game finished (room entering intermission).{RESET}", file=stderr)
                    if not follow:
                        _print_summary(total_events, start_time, errors, validate, output_json)
                        return
                elif room_status == "closed":
                    if not output_json:
                        print(f"\n{YELLOW}Room is closed (all agents left).{RESET}", file=stderr)
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
    status = data.get("status", "?")
    turn = data.get("turn", "?")
    events = data.get("events", [])

    color = CYAN if status == "active" else GREEN if status == "closed" else YELLOW if status in ("intermission", "ready_check") else CYAN
    print(f"\n{BOLD}{color}[{ts()}] Turn {turn} | {status.upper()}{RESET}")

    # Room lifecycle indicators
    if status == "ready_check":
        print(f"  {YELLOW}⏳ Waiting for agents to ready up...{RESET}")
    elif status == "intermission":
        game_count = data.get("game_count", "?")
        print(f"  {GREEN}🏁 Game #{game_count} complete. Awaiting ready for next game.{RESET}")

    for ev in events:
        visibility = ev.get("visibility", "public")
        msg = ev.get("message", "")
        color = GRAY if visibility != "public" else RESET
        print(f"  {color}» {msg}{RESET}")

    if data.get("game_over"):
        result = data.get("result") or {}
        winner_team = result.get("winner_team", "")
        winner_ids = result.get("winner_ids", [])
        if winner_team:
            print(f"  {BOLD}{GREEN}Winner: {winner_team.upper()}{RESET}")
        elif winner_ids:
            print(f"  {BOLD}{GREEN}Winner IDs: {winner_ids}{RESET}")


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
