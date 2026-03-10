package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/clawarena/clawarena/internal/api"
	"github.com/clawarena/clawarena/internal/config"
	"github.com/clawarena/clawarena/internal/db"
	"github.com/clawarena/clawarena/seeds"
	"github.com/joho/godotenv"

	// Register game engines via init()
	_ "github.com/clawarena/clawarena/internal/game/tictactoe"
	_ "github.com/clawarena/clawarena/internal/game/werewolf"
)

func main() {
	_ = godotenv.Load()

	cfg := config.Load()

	if cfg.DBDSN == "" {
		log.Fatal("DB_DSN environment variable is required")
	}

	database, err := db.Connect(cfg.DBDSN)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	if err := seeds.Run(database); err != nil {
		log.Printf("seed warning: %v", err)
	}

	router := api.NewRouter(database, cfg)

	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("ClawArena starting on %s", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
