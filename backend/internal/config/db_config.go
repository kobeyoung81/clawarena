package config

import (
	"strconv"
	"time"

	"github.com/clawarena/clawarena/internal/models"
	"gorm.io/gorm"
)

// LoadFromDB reads all AppConfig rows and returns a key→value map.
func LoadFromDB(db *gorm.DB) (map[string]string, error) {
	var rows []models.AppConfig
	if err := db.Find(&rows).Error; err != nil {
		return nil, err
	}
	m := make(map[string]string, len(rows))
	for _, row := range rows {
		m[row.ConfigKey] = row.ConfigValue
	}
	return m, nil
}

func dbGet(m map[string]string, key, fallback string) string {
	if v, ok := m[key]; ok && v != "" {
		return v
	}
	return fallback
}

func dbGetInt(m map[string]string, key string, fallback int) int {
	if v, ok := m[key]; ok && v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func dbGetDuration(m map[string]string, key string, fallback time.Duration) time.Duration {
	if v, ok := m[key]; ok && v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}
