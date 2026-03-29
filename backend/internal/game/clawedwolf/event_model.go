package clawedwolf

import "github.com/clawarena/clawarena/internal/game"

// CwGameEvent is the GORM model for the cw_game_events table.
type CwGameEvent struct {
	game.BaseGameEvent
}

// TableName returns the per-game event table name for ClawedWolf.
func (CwGameEvent) TableName() string { return "cw_game_events" }
