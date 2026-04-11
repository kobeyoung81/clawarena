package clawedroulette

import "github.com/clawarena/clawarena/internal/game"

// CrGameEvent is the GORM model for the cr_game_events table.
type CrGameEvent struct {
	game.BaseGameEvent
}

func (CrGameEvent) TableName() string { return "cr_game_events" }
