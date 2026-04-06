package game

import (
	"encoding/json"
	"time"
)

// PendingAction describes an action expected from a specific player.
type PendingAction struct {
	PlayerID     uint   `json:"player_id"`
	ActionType   string `json:"action_type"`
	Prompt       string `json:"prompt"`
	ValidTargets []int  `json:"valid_targets,omitempty"`
}

// EventEntity represents the actor (subject) or target (recipient) of a game event.
// Stored as a JSON column in the per-game event table for flexibility.
type EventEntity struct {
	AgentID *uint  `json:"agent_id,omitempty"` // nil for system-only events
	Seat    *int   `json:"seat,omitempty"`     // player seat number
	Team    string `json:"team,omitempty"`     // e.g., "wolf", "villager"
	Role    string `json:"role,omitempty"`     // e.g., "seer", "guard" (used in reveals)
}

// GameEvent is the fundamental unit of game progression.
// It captures "what happened" (structured actor/verb/target) and the resulting state.
// The EventType field doubles as the i18n translation key on the frontend.
// There is NO message field — the frontend formats display text from actor/target/details.
type GameEvent struct {
	Source     string          `json:"source"`              // "agent" or "system"
	EventType string          `json:"event_type"`          // verb / i18n key: "move", "vote", "phase_change", etc.
	Actor     *EventEntity    `json:"actor,omitempty"`     // who did it (nil for system-only events)
	Target    *EventEntity    `json:"target,omitempty"`    // who it was done to (nil if no target)
	Details   json.RawMessage `json:"details,omitempty"`   // event-type-specific context
	StateAfter json.RawMessage `json:"state_after"`        // full game state AFTER this event
	Visibility string         `json:"visibility"`          // "public", "player:<id>", "team:<name>"
	GameOver  bool            `json:"game_over"`
	Result    *GameResult     `json:"result,omitempty"`
}

// GameResult contains the outcome of a completed game.
type GameResult struct {
	WinnerIDs  []uint       `json:"winner_ids"`
	WinnerTeam string       `json:"winner_team,omitempty"`
	Scores     map[uint]int `json:"scores,omitempty"`
	TrophyURL  string       `json:"trophy_url,omitempty"`
}

// ApplyResult is returned by GameEngine.ApplyAction.
// It contains one or more events in order (agent action + system chain reactions).
// The final game state is Events[len(Events)-1].StateAfter.
type ApplyResult struct {
	Events []GameEvent `json:"events"`
}

// PhaseTimeout describes the current timeout expectation for the active game phase.
// Returned by GameEngine.GetPhaseTimeout. Nil means no active timeout.
type PhaseTimeout struct {
	Deadline time.Time `json:"deadline"`
	Label    string    `json:"label"` // e.g., "night_clawedwolf", "day_vote"
}

// GameEngine is the pluggable interface every game type must implement.
type GameEngine interface {
	// Syncronym returns the short code for this game type (e.g., "ttt", "cw").
	// Used as the prefix for per-game database tables (e.g., "ttt_game_events").
	Syncronym() string

	// InitState creates the initial game state and returns seed events
	// (e.g., "game_start", "roles_assigned").
	InitState(config json.RawMessage, players []uint) (json.RawMessage, []GameEvent, error)

	// GetPlayerView filters the full state to what a specific player should see.
	GetPlayerView(state json.RawMessage, playerID uint) (json.RawMessage, error)

	// GetSpectatorView filters the full state to what a public spectator should see.
	GetSpectatorView(state json.RawMessage) (json.RawMessage, error)

	// GetGodView returns the full unfiltered state (for admin/debug).
	GetGodView(state json.RawMessage) (json.RawMessage, error)

	// GetPendingActions returns actions currently expected from players.
	GetPendingActions(state json.RawMessage) ([]PendingAction, error)

	// ApplyAction processes a player's action and returns resulting events.
	// A single action may produce multiple events (agent action + system chain reactions).
	ApplyAction(state json.RawMessage, playerID uint, action json.RawMessage) (*ApplyResult, error)

	// GetPhaseTimeout returns the timeout for the current game phase, or nil if none.
	GetPhaseTimeout(state json.RawMessage) *PhaseTimeout

	// NewEventModel returns a new instance of this game's GORM event model.
	// Used for auto-migration and database queries.
	NewEventModel() GameEventRecord
}

// EngineEntry holds a game engine alongside its syncronym for lookup.
type EngineEntry struct {
	Engine    GameEngine
	Syncronym string
}

// Registry maps game type names (e.g., "tic_tac_toe") to their engine entries.
var Registry = map[string]*EngineEntry{}

// SyncronymIndex maps syncronyms (e.g., "ttt") to game type names.
var SyncronymIndex = map[string]string{}

// Register adds a game engine to the global registry.
// Called from init() functions in each game package.
// Panics if the syncronym is already registered (prevents duplicates).
func Register(name string, engine GameEngine) {
	syn := engine.Syncronym()
	if existing, exists := SyncronymIndex[syn]; exists {
		panic("duplicate game syncronym " + syn + ": " + name + " conflicts with " + existing)
	}
	Registry[name] = &EngineEntry{Engine: engine, Syncronym: syn}
	SyncronymIndex[syn] = name
}

// GetEngine returns the GameEngine for a game type name, or nil if not found.
func GetEngine(name string) GameEngine {
	if entry, ok := Registry[name]; ok {
		return entry.Engine
	}
	return nil
}
