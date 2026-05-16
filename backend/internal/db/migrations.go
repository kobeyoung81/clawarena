package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"

	mysql "github.com/go-sql-driver/mysql"
	"github.com/golang-migrate/migrate/v4"
	migratemysql "github.com/golang-migrate/migrate/v4/database/mysql"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	"github.com/clawarena/clawarena/internal/game"
	sqlmigrations "github.com/clawarena/clawarena/migrations"
)

const migrationVersionInitialSchema uint = 1

type MigrationStatus struct {
	Version        *uint
	Dirty          bool
	LegacyBaseline *uint
}

func EnsureMigrations(ctx context.Context, dsn string) error {
	migrator, migrationDB, err := OpenMigrator(dsn)
	if err != nil {
		return err
	}
	defer closeMigrator(migrator, migrationDB)

	if _, err := adoptLegacySchema(ctx, migrationDB, migrator); err != nil {
		return err
	}

	if err := migrator.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("apply migrations: %w", err)
	}

	if err := ensureRegisteredEventTablesPresent(ctx, migrationDB); err != nil {
		return err
	}

	return nil
}

func OpenMigrator(dsn string) (*migrate.Migrate, *sql.DB, error) {
	migrationDB, err := openMigrationDB(dsn)
	if err != nil {
		return nil, nil, err
	}

	driver, err := migratemysql.WithInstance(migrationDB, &migratemysql.Config{})
	if err != nil {
		_ = migrationDB.Close()
		return nil, nil, fmt.Errorf("create mysql migration driver: %w", err)
	}

	source, err := iofs.New(sqlmigrations.Files, ".")
	if err != nil {
		_ = migrationDB.Close()
		return nil, nil, fmt.Errorf("open embedded migrations: %w", err)
	}

	migrator, err := migrate.NewWithInstance("iofs", source, "mysql", driver)
	if err != nil {
		_ = migrationDB.Close()
		return nil, nil, fmt.Errorf("create migrator: %w", err)
	}

	return migrator, migrationDB, nil
}

func CurrentMigrationStatus(ctx context.Context, dsn string) (MigrationStatus, error) {
	migrator, migrationDB, err := OpenMigrator(dsn)
	if err != nil {
		return MigrationStatus{}, err
	}
	defer closeMigrator(migrator, migrationDB)

	version, dirty, err := migrator.Version()
	if err == nil {
		resolvedVersion := version
		return MigrationStatus{
			Version: &resolvedVersion,
			Dirty:   dirty,
		}, nil
	}
	if !errors.Is(err, migrate.ErrNilVersion) {
		return MigrationStatus{}, fmt.Errorf("read migration version: %w", err)
	}

	legacyBaseline, err := detectLegacyBaseline(ctx, migrationDB)
	if err != nil {
		return MigrationStatus{}, err
	}

	return MigrationStatus{
		LegacyBaseline: legacyBaseline,
	}, nil
}

func adoptLegacySchema(ctx context.Context, sqlDB *sql.DB, migrator *migrate.Migrate) (*uint, error) {
	version, dirty, err := migrator.Version()
	if err == nil {
		if dirty {
			return nil, fmt.Errorf("database migration state is dirty at version %d", version)
		}
		return nil, nil
	}
	if !errors.Is(err, migrate.ErrNilVersion) {
		return nil, fmt.Errorf("read migration version: %w", err)
	}

	baseline, err := detectLegacyBaseline(ctx, sqlDB)
	if err != nil {
		return nil, err
	}
	if baseline != nil {
		if err := migrator.Force(int(*baseline)); err != nil {
			return nil, fmt.Errorf("record legacy migration baseline %d: %w", *baseline, err)
		}
		return baseline, nil
	}

	return nil, nil
}

func detectLegacyBaseline(ctx context.Context, sqlDB *sql.DB) (*uint, error) {
	tables, err := listTables(ctx, sqlDB)
	if err != nil {
		return nil, err
	}

	return legacyBaselineFromTables(tables)
}

func legacyBaselineFromTables(tables map[string]bool) (*uint, error) {
	managedTables := allManagedLegacyTables()
	hasAllManagedTables := hasAllTables(tables, managedTables...)
	hasAnyManagedTables := hasAnyTables(tables, managedTables...)

	if hasAllManagedTables {
		return uintPtr(migrationVersionInitialSchema), nil
	}
	if hasAnyManagedTables {
		return nil, fmt.Errorf(
			"detected partial legacy schema without migration tracking; present tables: %s; missing tables: %s",
			strings.Join(sortedPresentTables(tables, managedTables), ", "),
			missingTables(tables, managedTables),
		)
	}

	return nil, nil
}

func listTables(ctx context.Context, sqlDB *sql.DB) (map[string]bool, error) {
	rows, err := sqlDB.QueryContext(ctx, "SELECT table_name FROM information_schema.tables WHERE table_schema = DATABASE()")
	if err != nil {
		return nil, fmt.Errorf("list database tables: %w", err)
	}
	defer rows.Close()

	tables := make(map[string]bool)
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return nil, fmt.Errorf("scan database table name: %w", err)
		}
		tables[tableName] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate database tables: %w", err)
	}

	return tables, nil
}

func allManagedLegacyTables() []string {
	tables := []string{
		"app_configs",
		"activity_events",
		"agents",
		"game_types",
		"languages",
		"rooms",
		"room_agents",
		"games",
		"game_players",
		"cr_game_events",
		"cw_game_events",
		"ttt_game_events",
	}

	sort.Strings(tables)
	return tables
}

func ensureRegisteredEventTablesPresent(ctx context.Context, sqlDB *sql.DB) error {
	tables, err := listTables(ctx, sqlDB)
	if err != nil {
		return err
	}

	missing := missingTableNames(tables, registeredEventTableNames())
	if len(missing) == 0 {
		return nil
	}

	return fmt.Errorf(
		"registered game event tables missing after migrations: %s; add a SQL migration before registering new engines",
		strings.Join(missing, ", "),
	)
}

func registeredEventTableNames() []string {
	tables := make([]string, 0, len(game.Registry))
	seen := make(map[string]struct{}, len(game.Registry))
	for _, entry := range game.Registry {
		table := entry.Engine.NewEventModel().TableName()
		if _, ok := seen[table]; ok {
			continue
		}
		seen[table] = struct{}{}
		tables = append(tables, table)
	}
	sort.Strings(tables)
	return tables
}

func hasAllTables(tables map[string]bool, names ...string) bool {
	for _, name := range names {
		if !tables[name] {
			return false
		}
	}
	return true
}

func hasAnyTables(tables map[string]bool, names ...string) bool {
	for _, name := range names {
		if tables[name] {
			return true
		}
	}
	return false
}

func missingTables(tables map[string]bool, names []string) string {
	return strings.Join(missingTableNames(tables, names), ", ")
}

func missingTableNames(tables map[string]bool, names []string) []string {
	missing := make([]string, 0, len(names))
	for _, name := range names {
		if !tables[name] {
			missing = append(missing, name)
		}
	}
	sort.Strings(missing)
	return missing
}

func sortedPresentTables(tables map[string]bool, names []string) []string {
	present := make([]string, 0, len(names))
	for _, name := range names {
		if tables[name] {
			present = append(present, name)
		}
	}
	sort.Strings(present)
	return present
}

func openMigrationDB(dsn string) (*sql.DB, error) {
	cfg, err := mysql.ParseDSN(strings.TrimSpace(dsn))
	if err != nil {
		return nil, fmt.Errorf("parse mysql dsn for migrations: %w", err)
	}
	cfg.MultiStatements = true

	migrationDB, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return nil, fmt.Errorf("open mysql migration connection: %w", err)
	}
	if err := migrationDB.Ping(); err != nil {
		_ = migrationDB.Close()
		return nil, fmt.Errorf("ping mysql migration connection: %w", err)
	}

	return migrationDB, nil
}

func closeMigrator(migrator *migrate.Migrate, sqlDB *sql.DB) {
	if migrator == nil {
		if sqlDB != nil {
			_ = sqlDB.Close()
		}
		return
	}
	_, _ = migrator.Close()
	if sqlDB != nil {
		_ = sqlDB.Close()
	}
}

func uintPtr(value uint) *uint {
	return &value
}
