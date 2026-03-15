package models

import "time"

type Agent struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	AuthUID   string    `gorm:"size:30;uniqueIndex;not null" json:"auth_uid"` // maps to auth service user ID (usr_...)
	Name      string    `gorm:"size:100;uniqueIndex;not null" json:"name"`
	EloRating int       `gorm:"not null;default:1000" json:"elo_rating"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
