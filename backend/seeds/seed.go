package seeds

import (
	_ "embed"
	"encoding/json"

	"github.com/clawarena/clawarena/internal/config"
	"github.com/clawarena/clawarena/internal/game"
	"github.com/clawarena/clawarena/internal/models"
	"gorm.io/gorm"
)

//go:embed rules/tic_tac_toe.md
var tttRules string

//go:embed rules/clawedwolf.md
var clawedwolfRules string

//go:embed rules/clawed_roulette.md
var clawedRouletteRules string

func Run(db *gorm.DB) error {
	if err := config.SeedDefaults(db); err != nil {
		return err
	}
	if err := seedLanguages(db); err != nil {
		return err
	}
	if err := seedGames(db); err != nil {
		return err
	}
	return syncSyncronyms(db)
}

func seedLanguages(db *gorm.DB) error {
	langs := []models.Language{
		{Code: "en", NativeName: "English", SortOrder: 1},
		{Code: "zh", NativeName: "中文", SortOrder: 2},
	}
	for _, l := range langs {
		db.Where("code = ?", l.Code).FirstOrCreate(&l)
	}
	return nil
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
			Name:        "clawedwolf",
			Description: "狼人杀 — 6-player social deduction game with hidden roles",
			MinPlayers:  6,
			MaxPlayers:  6,
			Config:      mustJSON(map[string]any{"roles": map[string]int{"clawedwolf": 2, "seer": 1, "guard": 1, "villager": 2}}),
			Rules:       clawedwolfRules,
		},
		{
			Name:        "clawed_roulette",
			Description: "Survival bluffing game — 2-4 players take turns firing a pistol loaded with live and blank rounds",
			MinPlayers:  2,
			MaxPlayers:  4,
			Config:      mustJSON(map[string]any{"total_bullets": 12, "max_hits": 2, "gadgets_per_player": 2}),
			Rules:       clawedRouletteRules,
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

// syncSyncronyms updates game_types.syncronym from the engine registry.
func syncSyncronyms(db *gorm.DB) error {
	for name, entry := range game.Registry {
		if err := db.Model(&models.GameType{}).
			Where("name = ?", name).
			Update("syncronym", entry.Syncronym).Error; err != nil {
			return err
		}
	}
	return nil
}
