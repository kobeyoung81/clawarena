package db

import (
	"reflect"
	"testing"

	_ "github.com/clawarena/clawarena/internal/game/clawedroulette"
	_ "github.com/clawarena/clawarena/internal/game/clawedwolf"
	_ "github.com/clawarena/clawarena/internal/game/tictactoe"
)

func TestLegacyBaselineFromTablesFreshDatabase(t *testing.T) {
	baseline, err := legacyBaselineFromTables(map[string]bool{})
	if err != nil {
		t.Fatalf("legacyBaselineFromTables returned error: %v", err)
	}
	if baseline != nil {
		t.Fatalf("expected no baseline for fresh database, got %v", *baseline)
	}
}

func TestLegacyBaselineFromTablesCompleteLegacySchema(t *testing.T) {
	tables := make(map[string]bool)
	for _, table := range allManagedLegacyTables() {
		tables[table] = true
	}

	baseline, err := legacyBaselineFromTables(tables)
	if err != nil {
		t.Fatalf("legacyBaselineFromTables returned error: %v", err)
	}
	if baseline == nil || *baseline != migrationVersionInitialSchema {
		t.Fatalf("expected baseline %d, got %v", migrationVersionInitialSchema, baseline)
	}
}

func TestLegacyBaselineFromTablesPartialLegacySchema(t *testing.T) {
	tables := map[string]bool{
		"agents":      true,
		"app_configs": true,
		"rooms":       true,
	}

	baseline, err := legacyBaselineFromTables(tables)
	if err == nil {
		t.Fatalf("expected error for partial legacy schema, got baseline %v", baseline)
	}
}

func TestAllManagedLegacyTablesStableSnapshot(t *testing.T) {
	expected := []string{
		"activity_events",
		"agents",
		"app_configs",
		"cr_game_events",
		"cw_game_events",
		"game_players",
		"game_types",
		"games",
		"languages",
		"room_agents",
		"rooms",
		"ttt_game_events",
	}

	if actual := allManagedLegacyTables(); !reflect.DeepEqual(actual, expected) {
		t.Fatalf("expected stable legacy baseline tables %v, got %v", expected, actual)
	}
}

func TestRegisteredEventTableNames(t *testing.T) {
	expected := []string{"cr_game_events", "cw_game_events", "ttt_game_events"}

	if actual := registeredEventTableNames(); !reflect.DeepEqual(actual, expected) {
		t.Fatalf("expected registered event tables %v, got %v", expected, actual)
	}
}

func TestMissingTableNames(t *testing.T) {
	tables := map[string]bool{
		"cw_game_events":  true,
		"ttt_game_events": true,
	}

	missing := missingTableNames(tables, registeredEventTableNames())
	expected := []string{"cr_game_events"}
	if !reflect.DeepEqual(missing, expected) {
		t.Fatalf("expected missing tables %v, got %v", expected, missing)
	}
}
