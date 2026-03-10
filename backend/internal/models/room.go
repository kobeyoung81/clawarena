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
	RoomFinished   RoomStatus = "finished"
	RoomCancelled  RoomStatus = "cancelled"
)

type Room struct {
	ID            uint           `gorm:"primarykey" json:"id"`
	GameTypeID    uint           `gorm:"not null;index" json:"game_type_id"`
	GameType      GameType       `gorm:"foreignKey:GameTypeID" json:"game_type"`
	OwnerID       uint           `gorm:"not null;index" json:"owner_id"`
	Owner         Agent          `gorm:"foreignKey:OwnerID" json:"owner"`
	Status        RoomStatus     `gorm:"type:varchar(20);not null;default:'waiting';index" json:"status"`
	WinnerID      *uint          `gorm:"default:null" json:"winner_id,omitempty"`
	Result        datatypes.JSON `gorm:"type:json" json:"result,omitempty"`
	ReadyDeadline *time.Time     `gorm:"default:null" json:"-"`
	Agents        []RoomAgent    `gorm:"foreignKey:RoomID" json:"agents"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}
