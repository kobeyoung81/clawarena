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
	AuthJWKSURL          string
	AuthPublicKeyContent string
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
	cfg.AuthJWKSURL = dbGet(m, "auth_jwks_url", "https://losclaws.com/.well-known/jwks.json")
	cfg.AuthPublicKeyContent = dbGet(m, "auth_public_key_content", "")

	return nil
}

// SeedDefaults inserts default AppConfig rows if they don't already exist.
func SeedDefaults(db *gorm.DB) error {
	defaults := []models.AppConfig{
		{ConfigKey: "port", ConfigValue: "8080", Description: "HTTP server port", Public: false},
		{ConfigKey: "frontend_url", ConfigValue: "http://localhost:5173", Description: "Frontend origin for CORS", Public: false},
		{ConfigKey: "auth_jwks_url", ConfigValue: "https://losclaws.com/.well-known/jwks.json", Description: "ClawAuth JWKS endpoint for JWT validation", Public: false},
		{ConfigKey: "auth_public_key_content", ConfigValue: "", Description: "PEM-encoded RSA public key content (dev/testing alternative to JWKS URL)", Public: false},
		{ConfigKey: "room_wait_timeout", ConfigValue: "10m", Description: "Duration before stale waiting rooms are cancelled", Public: true},
		{ConfigKey: "turn_timeout", ConfigValue: "60s", Description: "Agent turn timeout (reserved)", Public: true},
		{ConfigKey: "ready_check_timeout", ConfigValue: "20s", Description: "Ready-check countdown duration", Public: true},
		{ConfigKey: "rate_limit", ConfigValue: "60", Description: "Requests per minute per JWT identity", Public: false},
		{ConfigKey: "elo_k_factor", ConfigValue: "32", Description: "Elo rating K-factor for rank updates", Public: true},
		{ConfigKey: "auth_base_url", ConfigValue: "https://losclaws.com", Description: "ClawAuth service URL for browser auth checks (/auth/v1/humans/me, token refresh, logout)", Public: true},
		{ConfigKey: "portal_base_url", ConfigValue: "https://losclaws.com", Description: "Portal frontend URL for sign-in and user profile links", Public: true},
		{ConfigKey: "clawauth_skill_url", ConfigValue: "https://losclaws.com/skill/SKILL.md", Description: "ClawAuth skill URL for agent installation instructions", Public: true},
		{ConfigKey: "clawarena_skill_url", ConfigValue: "https://arena.losclaws.com/skill/SKILL.md", Description: "ClawArena skill URL for agent installation instructions", Public: true},
	}

	for i := range defaults {
		row := defaults[i]
		var existing models.AppConfig
		if err := db.First(&existing, "config_key = ?", row.ConfigKey).Error; err == nil {
			continue // already exists — preserve any manual edits
		}
		if err := db.Create(&row).Error; err != nil {
			return fmt.Errorf("seeding config key %q: %w", row.ConfigKey, err)
		}
	}
	return nil
}
