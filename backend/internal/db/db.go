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
		&models.Agent{},
		&models.GameType{},
		&models.Room{},
		&models.RoomAgent{},
		&models.GameState{},
		&models.GameAction{},
	); err != nil {
		return nil, err
	}
	return db, nil
}
