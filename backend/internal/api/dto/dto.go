package dto

import (
	"encoding/json"
	"time"
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

type GameEventDTO struct {
	Type       string `json:"type"`
	Message    string `json:"message"`
	Visibility string `json:"visibility"`
}

type GameResultDTO struct {
	WinnerIDs  []uint       `json:"winner_ids"`
	WinnerTeam string       `json:"winner_team,omitempty"`
	Scores     map[uint]int `json:"scores,omitempty"`
}

type ActionResponse struct {
	Events   []GameEventDTO `json:"events"`
	GameOver bool           `json:"game_over"`
	Result   *GameResultDTO `json:"result,omitempty"`
}

type GameStateResponse struct {
	RoomID        uint              `json:"room_id"`
	Status        string            `json:"status"`
	Turn          uint              `json:"turn"`
	CurrentAgentID *uint            `json:"current_agent_id,omitempty"`
	State         json.RawMessage   `json:"state"`
	PendingAction *PendingActionDTO `json:"pending_action,omitempty"`
	Agents        []RoomAgentInfo   `json:"agents"`
}

type HistoryPlayer struct {
	Seat    *int   `json:"seat,omitempty"`
	Slot    *uint8 `json:"slot,omitempty"`
	AgentID uint   `json:"agent_id"`
	Name    string `json:"name"`
	Role    string `json:"role,omitempty"`
}

type HistoryEntry struct {
	Turn      uint                   `json:"turn"`
	AgentID   *uint                  `json:"agent_id,omitempty"`
	Action    json.RawMessage        `json:"action,omitempty"`
	State     json.RawMessage        `json:"state"`
	Events    []GameEventDTO         `json:"events"`
	CreatedAt time.Time              `json:"created_at"`
}

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
