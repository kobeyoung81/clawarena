package integration

import (
	"database/sql"
	"fmt"
	"net/http/httptest"
	"os"
	"testing"

	_ "github.com/go-sql-driver/mysql"

	"github.com/clawarena/clawarena/internal/api"
	"github.com/clawarena/clawarena/internal/config"
	"github.com/clawarena/clawarena/internal/db"
	_ "github.com/clawarena/clawarena/internal/game/clawedroulette"
	_ "github.com/clawarena/clawarena/internal/game/tictactoe"
	_ "github.com/clawarena/clawarena/internal/game/clawedwolf"
	"github.com/clawarena/clawarena/seeds"
	"gorm.io/gorm"
)

var (
	server  *httptest.Server
	baseURL string
	testDB  *gorm.DB
)

func TestMain(m *testing.M) {
	if os.Getenv("CLAWARENA_INTEGRATION") != "1" {
		fmt.Println("Skipping integration tests (set CLAWARENA_INTEGRATION=1 to run)")
		os.Exit(0)
	}

	dsn := os.Getenv("TEST_DB_DSN")
	if dsn == "" {
		fmt.Fprintln(os.Stderr, "FATAL: TEST_DB_DSN is required")
		os.Exit(1)
	}

	// Extract database name from DSN to drop/create it.
	// DSN format: user:pass@tcp(host:port)/dbname?params
	dbName := extractDBName(dsn)
	adminDSN := dsn[:len(dsn)-len(dbName)-len("?charset=utf8mb4&parseTime=True&loc=Local")-1] // strip /dbname?params
	// More robust: rebuild admin DSN from parts before the last /
	adminDSN = dsnWithoutDB(dsn)

	// Reset test database
	rawDB, err := sql.Open("mysql", adminDSN+"/")
	if err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: cannot connect to MySQL: %v\n", err)
		os.Exit(1)
	}
	for _, stmt := range []string{
		fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", dbName),
		fmt.Sprintf("CREATE DATABASE `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci", dbName),
	} {
		if _, err := rawDB.Exec(stmt); err != nil {
			fmt.Fprintf(os.Stderr, "FATAL: %s: %v\n", stmt, err)
			os.Exit(1)
		}
	}
	rawDB.Close()

	// Connect via GORM (auto-migrates)
	testDB, err = db.Connect(dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: db.Connect: %v\n", err)
		os.Exit(1)
	}

	// Seed game types
	if err := seeds.Run(testDB); err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: seeds.Run: %v\n", err)
		os.Exit(1)
	}

	// Build config with high rate limit to avoid test flakiness
	cfg := &config.Config{
		FrontendURL:       "http://localhost:5173",
		RoomWaitTimeout:   0, // disable room timeout cleanup in tests
		TurnTimeout:       0,
		ReadyCheckTimeout: 20 * 1000_000_000, // 20s
		RateLimit:         10000,
	}

	// Start httptest server
	server = httptest.NewServer(api.NewRouter(testDB, cfg))
	baseURL = server.URL

	code := m.Run()

	// Cleanup
	server.Close()

	// Drop all tables
	sqlDB, _ := testDB.DB()
	if sqlDB != nil {
		sqlDB.Close()
	}

	os.Exit(code)
}

// extractDBName extracts the database name from a MySQL DSN.
func extractDBName(dsn string) string {
	// Find the last / before ?
	start := -1
	for i := len(dsn) - 1; i >= 0; i-- {
		if dsn[i] == '?' {
			// find / before this
			for j := i - 1; j >= 0; j-- {
				if dsn[j] == '/' {
					start = j + 1
					return dsn[start:i]
				}
			}
		}
	}
	// No ?, find last /
	for i := len(dsn) - 1; i >= 0; i-- {
		if dsn[i] == '/' {
			return dsn[i+1:]
		}
	}
	return dsn
}

// dsnWithoutDB returns the DSN prefix before the database name (e.g., "user:pass@tcp(host:port)").
func dsnWithoutDB(dsn string) string {
	// Find position of /dbname
	for i := len(dsn) - 1; i >= 0; i-- {
		if dsn[i] == '/' {
			// Check this isn't part of tcp()
			if i > 0 && dsn[i-1] == ')' {
				return dsn[:i]
			}
			// Keep looking
			return dsn[:i]
		}
	}
	return dsn
}
