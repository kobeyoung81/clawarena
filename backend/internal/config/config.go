package config

import (
	"fmt"
	"os"
	"time"

	"github.com/clawarena/clawarena/internal/models"
	"gorm.io/gorm"
)

type Config struct {
	Port              string
	DBDSN             string
	FrontendURL       string
	RoomWaitTimeout   time.Duration
	TurnTimeout       time.Duration
	ReadyCheckTimeout time.Duration
	RateLimit         int
	EloKFactor        float64
	AuthJWKSURL       string
	AuthPublicKeyPath string
}

// LoadInitial reads only DB_DSN from env (needed to connect to DB).
func LoadInitial() *Config {
	return &Config{
		DBDSN: os.Getenv("DB_DSN"),
	}
}

// LoadFromDB populates the config from the AppConfig table, falling back to defaults.
func (cfg *Config) LoadFromDB(db *gorm.DB) error {
	m, err := LoadFromDB(db)
	if err != nil {
		return fmt.Errorf("loading config from db: %w", err)
	}

	cfg.Port = dbGet(m, "port", "8080")
	cfg.FrontendURL = dbGet(m, "frontend_url", "http://localhost:5173")
	cfg.RoomWaitTimeout = dbGetDuration(m, "room_wait_timeout", 10*time.Minute)
	cfg.TurnTimeout = dbGetDuration(m, "turn_timeout", 60*time.Second)
	cfg.ReadyCheckTimeout = dbGetDuration(m, "ready_check_timeout", 20*time.Second)
	cfg.RateLimit = dbGetInt(m, "rate_limit", 60)
	cfg.EloKFactor = float64(dbGetInt(m, "elo_k_factor", 32))
	cfg.AuthJWKSURL = dbGet(m, "auth_jwks_url", "https://auth.losclaws.com/.well-known/jwks.json")
	cfg.AuthPublicKeyPath = dbGet(m, "auth_public_key_path", "")

	return nil
}

// SeedDefaults inserts default AppConfig rows if they don't already exist.
func SeedDefaults(db *gorm.DB) error {
	defaults := []models.AppConfig{
		{Key: "port", Value: "8080", Description: "HTTP server port", Public: false},
		{Key: "frontend_url", Value: "http://localhost:5173", Description: "Frontend origin for CORS", Public: false},
		{Key: "auth_jwks_url", Value: "https://auth.losclaws.com/.well-known/jwks.json", Description: "ClawAuth JWKS endpoint for JWT validation", Public: false},
		{Key: "auth_public_key_path", Value: "", Description: "Local RSA public key path (dev/testing alternative to JWKS URL)", Public: false},
		{Key: "room_wait_timeout", Value: "10m", Description: "Duration before stale waiting rooms are cancelled", Public: true},
		{Key: "turn_timeout", Value: "60s", Description: "Agent turn timeout (reserved)", Public: true},
		{Key: "ready_check_timeout", Value: "20s", Description: "Ready-check countdown duration", Public: true},
		{Key: "rate_limit", Value: "60", Description: "Requests per minute per JWT identity", Public: false},
		{Key: "elo_k_factor", Value: "32", Description: "Elo rating K-factor for rank updates", Public: true},
	}

	for i := range defaults {
		row := defaults[i]
		var existing models.AppConfig
		if err := db.First(&existing, "key = ?", row.Key).Error; err == nil {
			continue // already exists — preserve any manual edits
		}
		if err := db.Create(&row).Error; err != nil {
			return fmt.Errorf("seeding config key %q: %w", row.Key, err)
		}
	}
	return nil
}

