package models

import "time"

// AppConfig is a key-value store for all application configuration.
// Rows with Public=true are safe to return via the public config API.
type AppConfig struct {
	ConfigKey   string    `gorm:"column:config_key;primarykey;size:100" json:"config_key"`
	ConfigValue string    `gorm:"column:config_value;type:text" json:"config_value"`
	Description string    `gorm:"type:text" json:"description,omitempty"`
	Public      bool      `gorm:"not null;default:false" json:"public"`
	UpdatedAt   time.Time `json:"updated_at"`
}
