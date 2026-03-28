package models

import (
	"time"

	"gorm.io/datatypes"
)

type GameStatus string

const (
	GamePlaying  GameStatus = "playing"
	GameFinished GameStatus = "finished"
	GameAborted  GameStatus = "aborted"
)

// Game represents a single game session within a Room.
// A Room can host multiple Games over its lifetime.
type Game struct {
	ID         uint           `gorm:"primarykey" json:"id"`
	RoomID     uint           `gorm:"not null;index" json:"room_id"`
	Room       Room           `gorm:"foreignKey:RoomID" json:"-"`
	GameTypeID uint           `gorm:"not null;index" json:"game_type_id"`
	GameType   GameType       `gorm:"foreignKey:GameTypeID" json:"game_type"`
	Status     GameStatus     `gorm:"type:varchar(20);not null;default:'playing';index" json:"status"`
	WinnerID   *uint          `gorm:"default:null" json:"winner_id,omitempty"`
	Result     datatypes.JSON `gorm:"type:json" json:"result,omitempty"`
	StartedAt  time.Time      `json:"started_at"`
	FinishedAt *time.Time     `json:"finished_at,omitempty"`
}
