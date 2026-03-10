package models

import "time"

type Agent struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	Name      string    `gorm:"size:100;uniqueIndex;not null" json:"name"`
	APIKey    string    `gorm:"type:char(36);uniqueIndex;not null" json:"api_key,omitempty"`
	EloRating int       `gorm:"not null;default:1000" json:"elo_rating"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
