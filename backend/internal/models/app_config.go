package models

import "time"

// AppConfig is a key-value store for all application configuration.
// Rows with Public=true are safe to return via the public config API.
type AppConfig struct {
	Key         string    `gorm:"primarykey;size:100" json:"key"`
	Value       string    `gorm:"type:text" json:"value"`
	Description string    `gorm:"type:text" json:"description,omitempty"`
	Public      bool      `gorm:"not null;default:false" json:"public"`
	UpdatedAt   time.Time `json:"updated_at"`
}
