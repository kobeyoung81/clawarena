package seeds

import (
	_ "embed"
	"encoding/json"

	"github.com/clawarena/clawarena/internal/config"
	"github.com/clawarena/clawarena/internal/models"
	"gorm.io/gorm"
)

//go:embed rules/tic_tac_toe.md
var tttRules string

//go:embed rules/werewolf.md
var werewolfRules string

func Run(db *gorm.DB) error {
	if err := config.SeedDefaults(db); err != nil {
		return err
	}
	return seedGames(db)
}

func seedGames(db *gorm.DB) error {
	games := []models.GameType{
		{
			Name:        "tic_tac_toe",
			Description: "Classic 3x3 Tic-Tac-Toe for 2 players",
			MinPlayers:  2,
			MaxPlayers:  2,
			Config:      mustJSON(map[string]any{"board_size": 3}),
			Rules:       tttRules,
		},
		{
			Name:        "werewolf",
			Description: "狼人杀 — 6-player social deduction game with hidden roles",
			MinPlayers:  6,
			MaxPlayers:  6,
			Config:      mustJSON(map[string]any{"roles": map[string]int{"werewolf": 2, "seer": 1, "guard": 1, "villager": 2}}),
			Rules:       werewolfRules,
		},
	}

	for i := range games {
		g := games[i]
		var existing models.GameType
		err := db.Where("name = ?", g.Name).First(&existing).Error
		if err == nil {
			continue // already seeded
		}
		if err := db.Create(&g).Error; err != nil {
			return err
		}
	}
	return nil
}

func mustJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}
