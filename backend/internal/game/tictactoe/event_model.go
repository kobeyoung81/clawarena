package tictactoe

import "github.com/clawarena/clawarena/internal/game"

// TttGameEvent is the GORM model for the ttt_game_events table.
type TttGameEvent struct {
	game.BaseGameEvent
}

// TableName returns the per-game event table name for Tic-Tac-Toe.
func (TttGameEvent) TableName() string { return "ttt_game_events" }
