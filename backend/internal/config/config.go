package config

import (
	"os"
	"time"
)

type Config struct {
	Port               string
	DBDSN              string
	FrontendURL        string
	RoomWaitTimeout    time.Duration
	TurnTimeout        time.Duration
	ReadyCheckTimeout  time.Duration
	RateLimit          int
}

func Load() *Config {
	return &Config{
		Port:              getEnv("PORT", "8080"),
		DBDSN:             getEnv("DB_DSN", ""),
		FrontendURL:       getEnv("FRONTEND_URL", "http://localhost:5173"),
		RoomWaitTimeout:   parseDuration(getEnv("ROOM_WAIT_TIMEOUT", "10m")),
		TurnTimeout:       parseDuration(getEnv("TURN_TIMEOUT", "60s")),
		ReadyCheckTimeout: parseDuration(getEnv("READY_CHECK_TIMEOUT", "20s")),
		RateLimit:         60,
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func parseDuration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0
	}
	return d
}
