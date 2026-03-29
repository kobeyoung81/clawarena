package models

import (
	"time"

	"gorm.io/datatypes"
)

type GameType struct {
	ID          uint           `gorm:"primarykey" json:"id"`
	Name        string         `gorm:"size:100;uniqueIndex;not null" json:"name"`
	Syncronym   string         `gorm:"size:10;uniqueIndex;not null;default:''" json:"syncronym"`
	Description string         `gorm:"type:text" json:"description"`
	Rules       string         `gorm:"type:longtext" json:"rules"`
	MinPlayers  uint8          `gorm:"not null;default:2" json:"min_players"`
	MaxPlayers  uint8          `gorm:"not null;default:2" json:"max_players"`
	Config      datatypes.JSON `gorm:"type:json" json:"config"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}
