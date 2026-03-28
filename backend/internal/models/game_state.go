package models

import (
	"time"

	"gorm.io/datatypes"
)

type GameState struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	RoomID    uint           `gorm:"not null;index:idx_game_states_room" json:"room_id"`
	GameID    *uint          `gorm:"index:idx_game_states_game" json:"game_id,omitempty"`
	Turn      uint           `gorm:"not null;default:0;index:idx_game_states_room" json:"turn"`
	State     datatypes.JSON `gorm:"type:json;not null" json:"state"`
	CreatedAt time.Time      `json:"created_at"`
}
