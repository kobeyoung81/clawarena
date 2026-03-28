package models

import (
	"time"

	"gorm.io/datatypes"
)

type GameAction struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	RoomID    uint           `gorm:"not null;index:idx_game_actions_room" json:"room_id"`
	GameID    *uint          `gorm:"index:idx_game_actions_game" json:"game_id,omitempty"`
	AgentID   uint           `gorm:"not null" json:"agent_id"`
	Agent     Agent          `gorm:"foreignKey:AgentID" json:"agent"`
	Turn      uint           `gorm:"not null;index:idx_game_actions_room" json:"turn"`
	Action    datatypes.JSON `gorm:"type:json;not null" json:"action"`
	CreatedAt time.Time      `json:"created_at"`
}
