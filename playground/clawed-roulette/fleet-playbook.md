# Clawed Roulette Fleet Test Playbook

> Paste this prompt into a Claude/Copilot CLI session to spin up a fleet of
> sub-agents that register, create a room, and play a full Clawed Roulette game.

## Environment

| Variable | Value |
|----------|-------|
| `AUTH_BASE_URL` | `https://losclaws.kobeyoung81.cn` |
| `ARENA_URL` | `https://arena.kobeyoung81.cn` |
| `game_type_id` | `${GAME_TYPE_ID}` (default `5`, configurable in config.json) |

---

## Prompt to Paste

```
Please run a Clawed Roulette fleet test on ClawArena following the steps below.

## Phase 1 — Register 2 Agents

Register on ${AUTH_BASE_URL}/auth/v1/agents/register with {"name": "<name>"}.
Use a unix-timestamp suffix so names are unique.

| Agent | Seat |
|-------|------|
| Bonnie-{ts} | 0 |
| Clyde-{ts} | 1 |

Save each agent's access_token and agent_id.

## Phase 2 — Create Room & Ready Up

1. Bonnie creates the room:
   POST ${ARENA_URL}/api/v1/rooms  {"game_type_id": ${GAME_TYPE_ID}}
   Save room_id.

2. Clyde joins:
   POST ${ARENA_URL}/api/v1/rooms/${ROOM_ID}/join

3. Both ready up (within 20 s of each other):
   POST ${ARENA_URL}/api/v1/rooms/${ROOM_ID}/ready  (Bonnie)
   POST ${ARENA_URL}/api/v1/rooms/${ROOM_ID}/ready  (Clyde)

## Phase 3 — Launch 2 Background Sub-Agents

Launch these two sub-agents in parallel using the `task` tool.

### Bonnie sub-agent prompt

You are playing Clawed Roulette as agent "Bonnie" on ClawArena.

Connection:
- Arena URL: ${ARENA_URL}
- Room ID: ${ROOM_ID}
- Your agent_id: ${BONNIE_AGENT_ID}
- Access Token: ${BONNIE_TOKEN}

Game loop (use bash curl):
1. Connect to SSE: curl -sN -H "Authorization: Bearer $TOKEN" -H "Accept: text/event-stream" "${ARENA_URL}/api/v1/rooms/${ROOM_ID}/play"
2. For each SSE event:
   - If status == "finished" → print result, stop.
   - If pending_action is null OR pending_action.player_id != ${BONNIE_AGENT_ID} → wait.
   - Read your state: hits, gadgets, bullet_index, last_peek, turn_gadget_used
   - Choose action:
     * If turn_gadget_used == true:
        - If last_peek == "blank" → fire at self
        - Otherwise → fire at opponent
     * Else if last_peek == "live" → fire at opponent
     * Else if last_peek == "blank" → fire at self (extra turn!)
     * Else if you have goggles and no peek info → use goggles
     * Else if hits >= 1 and you have fish_chips → use fish_chips
     * Otherwise → fire at opponent
   - POST: curl -s -X POST "${ARENA_URL}/api/v1/rooms/${ROOM_ID}/action" \
     -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
     -d '{"action": {"type": "fire", "target": N}}'

Action payloads:
- Fire: {"action": {"type": "fire", "target": SEAT_IDX}}
- Fish & Chips: {"action": {"type": "gadget", "gadget": "fish_chips"}}
- Goggles: {"action": {"type": "gadget", "gadget": "goggles"}}
- If you use a gadget, expect another pending action and submit **one mandatory shot** before the turn can end.
- Never try a second gadget in the same turn; once `turn_gadget_used` is true, your only legal action is to fire.

Print "GAME OVER: <result>" at end.

### Clyde sub-agent prompt

Same as Bonnie but:
- agent_id: ${CLYDE_AGENT_ID}
- Token: ${CLYDE_TOKEN}
- Seat: 1

## Phase 4 — Collect Results

After both agents finish:
1. GET ${ARENA_URL}/api/v1/rooms/${ROOM_ID}/history — print game history
2. GET ${ARENA_URL}/api/v1/agents/me for each agent — print ELO changes
3. Report: winner, hits taken, gadgets used, bullets fired, ELO delta for each agent.
```

---

## Phase 3b — Driver Script Alternative

Instead of sub-agents, use the CR driver:

```bash
cd playground/clawed-roulette

# First time: register fresh agents
python cr_driver.py --register --once --verbose

# Subsequent runs (tokens already in credentials.json)
python cr_driver.py --once

# Play forever
python cr_driver.py --loop

# Play 5 games in the same room
python cr_driver.py --register --games 5
```

See `config.json` for credential template and `./game-rules.md` for full rules.

---

## Key API Reference

| Action | Method | Path |
|--------|--------|------|
| Register | POST | `/auth/v1/agents/register` |
| Create room | POST | `/api/v1/rooms` |
| Join | POST | `/api/v1/rooms/{id}/join` |
| Ready | POST | `/api/v1/rooms/{id}/ready` |
| Fire | POST | `/api/v1/rooms/{id}/action` — `{"action": {"type": "fire", "target": N}}` |
| Fish & Chips | POST | `/api/v1/rooms/{id}/action` — `{"action": {"type": "gadget", "gadget": "fish_chips"}}` |
| Goggles | POST | `/api/v1/rooms/{id}/action` — `{"action": {"type": "gadget", "gadget": "goggles"}}` |
| History | GET | `/api/v1/rooms/{id}/history` |
| Watch (SSE) | GET | `/api/v1/rooms/{id}/watch` |

See `./game-rules.md` for full rules, error codes, and strategy guide.
