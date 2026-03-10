package game

import "encoding/json"

// PendingAction describes an action expected from a specific player.
type PendingAction struct {
	PlayerID     uint   `json:"player_id"`
	ActionType   string `json:"action_type"`
	Prompt       string `json:"prompt"`
	ValidTargets []int  `json:"valid_targets,omitempty"`
}

// GameEvent represents something that happened in the game.
type GameEvent struct {
	Type       string          `json:"type"`
	Message    string          `json:"message"`
	Visibility string          `json:"visibility"`
	Data       json.RawMessage `json:"data,omitempty"`
}

// GameResult contains the outcome of a completed game.
type GameResult struct {
	WinnerIDs  []uint       `json:"winner_ids"`
	WinnerTeam string       `json:"winner_team,omitempty"`
	Scores     map[uint]int `json:"scores,omitempty"`
}

// ActionResult is returned by ApplyAction.
type ActionResult struct {
	NewState json.RawMessage `json:"new_state"`
	Events   []GameEvent     `json:"events"`
	GameOver bool            `json:"game_over"`
	Result   *GameResult     `json:"result,omitempty"`
}

// GameEngine is the pluggable interface every game type must implement.
type GameEngine interface {
	InitState(config json.RawMessage, players []uint) (json.RawMessage, error)
	GetPlayerView(state json.RawMessage, playerID uint) (json.RawMessage, error)
	GetSpectatorView(state json.RawMessage) (json.RawMessage, error)
	GetGodView(state json.RawMessage) (json.RawMessage, error)
	GetPendingActions(state json.RawMessage) ([]PendingAction, error)
	ApplyAction(state json.RawMessage, playerID uint, action json.RawMessage) (ActionResult, error)
}

// Registry maps game type names to their engine implementations.
var Registry = map[string]GameEngine{}

func Register(name string, engine GameEngine) {
	Registry[name] = engine
}
