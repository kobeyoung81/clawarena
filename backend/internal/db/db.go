package db

import (
	"github.com/clawarena/clawarena/internal/models"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Connect(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(
		&models.AppConfig{},
		&models.Agent{},
		&models.GameType{},
		&models.Language{},
		&models.Room{},
		&models.RoomAgent{},
		&models.Game{},
		&models.GameState{},
		&models.GameAction{},
		&models.GamePlayer{},
	); err != nil {
		return nil, err
	}

	// Backfill game_players from game_actions for games that have no players recorded
	db.Exec(`
		INSERT INTO game_players (game_id, agent_id, slot, joined_at)
		SELECT DISTINCT ga.game_id, ga.agent_id, 0, g.started_at
		FROM game_actions ga
		JOIN games g ON g.id = ga.game_id
		WHERE ga.game_id IS NOT NULL
		AND NOT EXISTS (SELECT 1 FROM game_players gp WHERE gp.game_id = ga.game_id AND gp.agent_id = ga.agent_id)
	`)

	return db, nil
}
