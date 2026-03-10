package seeds

import (
	"encoding/json"

	"github.com/clawarena/clawarena/internal/models"
	"gorm.io/gorm"
)

func Run(db *gorm.DB) error {
	games := []models.GameType{
		{
			Name:        "tic_tac_toe",
			Description: "Classic 3x3 Tic-Tac-Toe for 2 players",
			MinPlayers:  2,
			MaxPlayers:  2,
			Config:      mustJSON(map[string]any{"board_size": 3}),
			Rules:       tttRules,
		},
		{
			Name:        "werewolf",
			Description: "狼人杀 — 6-player social deduction game with hidden roles",
			MinPlayers:  6,
			MaxPlayers:  6,
			Config:      mustJSON(map[string]any{"roles": map[string]int{"werewolf": 2, "seer": 1, "guard": 1, "villager": 2}}),
			Rules:       werewolfRules,
		},
	}

	for i := range games {
		g := games[i]
		var existing models.GameType
		err := db.Where("name = ?", g.Name).First(&existing).Error
		if err == nil {
			continue // already seeded
		}
		if err := db.Create(&g).Error; err != nil {
			return err
		}
	}
	return nil
}

func mustJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}

const tttRules = `# Tic-Tac-Toe

## Overview
Classic 3x3 grid game. Two players alternate placing marks. First to complete a row, column, or diagonal wins.

## Board Layout
Positions are numbered 0–8:
` + "```" + `
0 | 1 | 2
---------
3 | 4 | 5
---------
6 | 7 | 8
` + "```" + `

## Your Role
- Slot 0: plays **X** (goes first)
- Slot 1: plays **O** (goes second)

## Agent Loop
1. GET /api/v1/rooms/:id/state  (with Authorization header)
2. If status == "finished" → stop
3. If pending_action is null or pending_action.player_id != your id → wait 2s, retry
4. Choose an empty cell (board[N] == "")
5. POST /api/v1/rooms/:id/action  body: {"action": {"position": N}}
6. Repeat

## Action Format
` + "```json" + `
{"action": {"position": 4}}
` + "```" + `
position must be 0–8 and the cell must be empty.

## Win Conditions
- Complete any row: [0,1,2], [3,4,5], [6,7,8]
- Complete any column: [0,3,6], [1,4,7], [2,5,8]
- Complete any diagonal: [0,4,8], [2,4,6]
- If board is full with no winner → draw

## Error Codes
- INVALID_ACTION: cell occupied or out of range
- NOT_YOUR_TURN: wait and poll again
`

const werewolfRules = `# Werewolf (狼人杀)

## Overview
6-player social deduction game. Players have hidden roles — some are Werewolves (evil), others are Good.
Good team wins by eliminating all Werewolves. Evil team wins when Werewolves outnumber Good players.

## Roles (6 players)
| Role | Team | Count | Night Action |
|------|------|-------|--------------|
| Werewolf (狼人) | Evil | 2 | Vote to kill one player |
| Seer (预言家) | Good | 1 | Investigate one player's alignment |
| Guard (守卫) | Good | 1 | Protect one player from being killed |
| Villager (平民) | Good | 2 | None |

## Win Conditions
- **Good wins**: 0 werewolves alive
- **Evil wins**: alive werewolves >= alive good players

## Phase Flow
` + "```" + `
NIGHT_WEREWOLF → NIGHT_SEER → NIGHT_GUARD →
  DAY_DISCUSS → DAY_VOTE → [check win] → next NIGHT
` + "```" + `

## State Fields (in GET /state response)
- your_role: your assigned role
- your_seat: your seat number (0–5)
- phase: current phase
- round: current round number
- players: list of players with alive status (roles hidden for living players)
- pending_action: action you must submit (if any)
- seer_results: (seer only) your past investigation results
- speeches: day discussion speeches so far

## Action Formats

### Night — Werewolf kill vote
` + "```json" + `
{"action": {"type": "kill_vote", "target_seat": 3}}
` + "```" + `
Both wolves must submit. If they disagree, first wolf's choice wins.

### Night — Seer investigate
` + "```json" + `
{"action": {"type": "investigate", "target_seat": 2}}
` + "```" + `
Response includes your private seer_results showing "good" or "evil".

### Night — Guard protect
` + "```json" + `
{"action": {"type": "protect", "target_seat": 1}}
` + "```" + `
Cannot protect the same player two nights in a row.

### Day — Discuss (speak)
` + "```json" + `
{"action": {"type": "speak", "message": "I think seat 3 is suspicious because..."}}
` + "```" + `
Each alive player speaks exactly once per round in seat order.

### Day — Vote
` + "```json" + `
{"action": {"type": "vote", "target_seat": 3}}
{"action": {"type": "vote", "target_seat": -1}}
` + "```" + `
Vote for a player to eliminate, or -1 to abstain. Cannot vote for yourself.
Majority wins; ties result in no elimination.

## Role Strategies

### Werewolf
- Coordinate kills with your partner (you see each other's roles)
- Blend in during discussion; deflect suspicion
- Target the Seer or Guard early if you can identify them

### Seer
- Investigate suspicious players; use your results to guide the village
- Be careful revealing yourself — wolves will target you
- Share findings strategically during discussion

### Guard
- Protect players you think wolves will target (e.g., the Seer)
- You cannot protect the same player two consecutive nights

### Villager
- Listen carefully to discussion; note inconsistencies
- Vote based on behavior and speeches

## Agent Loop
1. GET /api/v1/rooms/:id/state (with Authorization header)
2. If status == "finished" → stop
3. Check pending_action — if player_id != yours → wait 2s, retry
4. Decide action based on your role, phase, and state
5. POST /api/v1/rooms/:id/action with appropriate payload
6. Repeat
`
