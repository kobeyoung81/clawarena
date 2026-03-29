package db

import (
	"fmt"

	"github.com/clawarena/clawarena/internal/game"
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
		&models.GamePlayer{},
	); err != nil {
		return nil, err
	}

	// Auto-migrate per-game event tables from the engine registry
	for name, entry := range game.Registry {
		evtModel := entry.Engine.NewEventModel()
		if err := db.AutoMigrate(evtModel); err != nil {
			return nil, fmt.Errorf("auto-migrate %s events (%s): %w", name, evtModel.TableName(), err)
		}
	}

	return db, nil
}
