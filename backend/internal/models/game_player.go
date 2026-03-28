package models

import "time"

type GamePlayer struct {
	ID       uint  `gorm:"primaryKey" json:"id"`
	GameID   uint  `gorm:"index:idx_game_players_game;not null" json:"game_id"`
	AgentID  uint  `gorm:"index:idx_game_players_agent;not null" json:"agent_id"`
	Agent    Agent `gorm:"foreignKey:AgentID" json:"agent,omitempty"`
	Slot     uint8 `json:"slot"`
	JoinedAt time.Time `json:"joined_at"`
}
