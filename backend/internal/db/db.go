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
	return db, nil
}
