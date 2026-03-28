package models

import (
	"time"

	"gorm.io/datatypes"
)

type RoomStatus string

const (
	RoomWaiting    RoomStatus = "waiting"
	RoomReadyCheck RoomStatus = "ready_check"
	RoomPlaying    RoomStatus = "playing"
	RoomPostGame   RoomStatus = "post_game"
	RoomFinished   RoomStatus = "finished"
	RoomCancelled  RoomStatus = "cancelled"
	RoomDead       RoomStatus = "dead"
)

type Room struct {
	ID            uint           `gorm:"primarykey" json:"id"`
	GameTypeID    uint           `gorm:"not null;index" json:"game_type_id"`
	GameType      GameType       `gorm:"foreignKey:GameTypeID" json:"game_type"`
	OwnerID       uint           `gorm:"not null;index" json:"owner_id"`
	Owner         Agent          `gorm:"foreignKey:OwnerID" json:"owner"`
	Status        RoomStatus     `gorm:"type:varchar(20);not null;default:'waiting';index" json:"status"`
	Language      string         `gorm:"size:10;not null;default:'en'" json:"language"`
	GameCount     int            `gorm:"not null;default:0" json:"game_count"`
	CurrentGameID *uint          `gorm:"default:null" json:"current_game_id,omitempty"`
	WinnerID      *uint          `gorm:"default:null" json:"winner_id,omitempty"`
	Result        datatypes.JSON `gorm:"type:json" json:"result,omitempty"`
	ReadyDeadline *time.Time     `gorm:"default:null" json:"-"`
	Agents        []RoomAgent    `gorm:"foreignKey:RoomID" json:"agents"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}
