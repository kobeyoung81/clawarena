package dto

import (
	"encoding/json"
	"time"

	"github.com/clawarena/clawarena/internal/game"
)

// Agent
type RegisterAgentRequest struct {
	Name string `json:"name"`
}

type AgentResponse struct {
	ID        uint      `json:"id"`
	Name      string    `json:"name"`
	EloRating int       `json:"elo_rating"`
	CreatedAt time.Time `json:"created_at"`
}

// Rooms
type CreateRoomRequest struct {
	GameTypeID uint   `json:"game_type_id"`
	Language   string `json:"language,omitempty"`
}

type RoomAgentInfo struct {
	ID      uint   `json:"id"`
	Name    string `json:"name"`
	Slot    uint8  `json:"slot"`
	Score   int    `json:"score"`
	Ready   bool   `json:"ready"`
	AgentID uint   `json:"agent_id"`
	Status  string `json:"status"`
}

type GameTypeInfo struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	MinPlayers  uint8  `json:"min_players"`
	MaxPlayers  uint8  `json:"max_players"`
}

type OwnerInfo struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

type RoomResponse struct {
	ID            uint            `json:"id"`
	GameType      GameTypeInfo    `json:"game_type"`
	Status        string          `json:"status"`
	Owner         OwnerInfo       `json:"owner"`
	Language      string          `json:"language"`
	GameCount     int             `json:"game_count"`
	CurrentGameID *uint           `json:"current_game_id,omitempty"`
	Agents        []RoomAgentInfo `json:"agents"`
	CreatedAt     time.Time       `json:"created_at"`
}

type JoinRoomResponse struct {
	Slot     uint8      `json:"slot"`
	Status   string     `json:"status"`
	Message  string     `json:"message"`
	Deadline *time.Time `json:"deadline,omitempty"`
}

type ReadyResponse struct {
	Status     string     `json:"status"`
	ReadyCount int        `json:"ready_count"`
	Total      int        `json:"total"`
	Deadline   *time.Time `json:"deadline,omitempty"`
	Message    string     `json:"message,omitempty"`
}

type LeaveResponse struct {
	Message string `json:"message"`
}

// Gameplay
type SubmitActionRequest struct {
	Action json.RawMessage `json:"action"`
}

type PendingActionDTO struct {
	PlayerID     uint   `json:"player_id"`
	ActionType   string `json:"action_type"`
	Prompt       string `json:"prompt"`
	ValidTargets []int  `json:"valid_targets,omitempty"`
}

// GameEventDTO is the event-sourced event format returned in action responses and SSE streams.
type GameEventDTO struct {
	Seq        uint              `json:"seq"`
	GameID     uint              `json:"game_id"`
	Source     string            `json:"source"`
	EventType  string            `json:"event_type"`
	Actor      *game.EventEntity `json:"actor,omitempty"`
	Target     *game.EventEntity `json:"target,omitempty"`
	Details    json.RawMessage   `json:"details,omitempty"`
	State      json.RawMessage   `json:"state"`
	Visibility string            `json:"visibility"`
	GameOver   bool              `json:"game_over"`
	Result     *GameResultDTO    `json:"result,omitempty"`
	CreatedAt  time.Time         `json:"created_at"`
}

type GameResultDTO struct {
	WinnerIDs  []uint       `json:"winner_ids"`
	WinnerTeam string       `json:"winner_team,omitempty"`
	Scores     map[uint]int `json:"scores,omitempty"`
}

// ActionResponse is returned from SubmitAction.
type ActionResponse struct {
	Events   []GameEventDTO `json:"events"`
	GameOver bool           `json:"game_over"`
	Result   *GameResultDTO `json:"result,omitempty"`
}

// EventHistoryResponse is the event-sourced history format.
type EventHistoryResponse struct {
	RoomID   uint           `json:"room_id"`
	GameID   uint           `json:"game_id"`
	Status   string         `json:"status"`
	GameType string         `json:"game_type"`
	Result   *GameResultDTO `json:"result,omitempty"`
	Players  []HistoryPlayer `json:"players"`
	Events   []GameEventDTO `json:"events"`
}

// SSEEventPayload is the enriched SSE event broadcast per game event.
type SSEEventPayload struct {
	Seq            uint              `json:"seq"`
	GameID         uint              `json:"game_id"`
	RoomID         uint              `json:"room_id"`
	Source         string            `json:"source"`
	EventType      string            `json:"event_type"`
	Actor          *game.EventEntity `json:"actor,omitempty"`
	Target         *game.EventEntity `json:"target,omitempty"`
	Details        json.RawMessage   `json:"details,omitempty"`
	State          json.RawMessage   `json:"state"`
	Visibility     string            `json:"visibility"`
	PendingAction  *PendingActionDTO `json:"pending_action,omitempty"`
	CurrentAgentID *uint             `json:"current_agent_id,omitempty"`
	Agents         []RoomAgentInfo   `json:"agents"`
	GameType       string            `json:"game_type"`
	GameOver       bool              `json:"game_over"`
	Result         *GameResultDTO    `json:"result,omitempty"`
	Status         string            `json:"status,omitempty"`
	Message        string            `json:"message,omitempty"`
}

type HistoryPlayer struct {
	Seat    *int   `json:"seat,omitempty"`
	Slot    *uint8 `json:"slot,omitempty"`
	AgentID uint   `json:"agent_id"`
	Name    string `json:"name"`
	Role    string `json:"role,omitempty"`
}

// HistoryEntry is kept for backward compatibility with the legacy history endpoint.
type HistoryEntry struct {
	Turn      uint            `json:"turn"`
	AgentID   *uint           `json:"agent_id,omitempty"`
	Action    json.RawMessage `json:"action,omitempty"`
	State     json.RawMessage `json:"state"`
	Events    []GameEventDTO  `json:"events"`
	CreatedAt time.Time       `json:"created_at"`
}

// HistoryResponse is the legacy room-based history format (kept for backward compat).
type HistoryResponse struct {
	RoomID   uint            `json:"room_id"`
	Status   string          `json:"status"`
	GameType string          `json:"game_type"`
	Result   *GameResultDTO  `json:"result,omitempty"`
	Players  []HistoryPlayer `json:"players"`
	Timeline []HistoryEntry  `json:"timeline"`
}

// Game history
type GamePlayerInfo struct {
	AgentID uint   `json:"agent_id"`
	Name    string `json:"name"`
	Slot    uint8  `json:"slot"`
}

type GameListItem struct {
	ID         uint             `json:"id"`
	RoomID     uint             `json:"room_id"`
	GameType   GameTypeInfo     `json:"game_type"`
	Status     string           `json:"status"`
	WinnerID   *uint            `json:"winner_id,omitempty"`
	Result     *GameResultDTO   `json:"result,omitempty"`
	Players    []GamePlayerInfo `json:"players"`
	StartedAt  time.Time        `json:"started_at"`
	FinishedAt *time.Time       `json:"finished_at,omitempty"`
}

type GameListResponse struct {
	Games      []GameListItem `json:"games"`
	TotalCount int64          `json:"total_count"`
	Page       int            `json:"page"`
	PerPage    int            `json:"per_page"`
}

// Error
type ErrorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}
