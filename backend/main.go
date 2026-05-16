package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/clawarena/clawarena/internal/api"
	"github.com/clawarena/clawarena/internal/config"
	"github.com/clawarena/clawarena/internal/db"
	"github.com/clawarena/clawarena/seeds"

	// Register game engines via init()
	_ "github.com/clawarena/clawarena/internal/game/clawedroulette"
	_ "github.com/clawarena/clawarena/internal/game/clawedwolf"
	_ "github.com/clawarena/clawarena/internal/game/tictactoe"
)

func main() {
	cfg := config.LoadInitial()

	if cfg.DBDSN == "" {
		cfg.DBDSN = os.Getenv("DB_DSN")
	}
	if cfg.DBDSN == "" {
		log.Fatal("DB_DSN environment variable is required")
	}

	database, err := db.Connect(cfg.DBDSN)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	if err := db.EnsureMigrations(context.Background(), cfg.DBDSN); err != nil {
		log.Fatalf("failed to apply migrations: %v", err)
	}

	if err := seeds.Run(database); err != nil {
		log.Printf("seed warning: %v", err)
	}

	if err := cfg.LoadFromDB(database); err != nil {
		log.Fatalf("failed to load config from database: %v", err)
	}

	router := api.NewRouter(database, cfg)

	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("ClawArena starting on %s", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
