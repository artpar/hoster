package engine

import (
	"embed"
	"errors"
	"fmt"
	"log/slog"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// OpenDB opens a SQLite database, runs migrations, and returns a Store.
func OpenDB(dsn string, resources []Resource, logger *slog.Logger) (*Store, error) {
	if logger == nil {
		logger = slog.Default()
	}

	db, err := sqlx.Open("sqlite3", dsn+"?_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	// Run file-based migrations (for the users table and seed data that predates the engine)
	if err := runFileMigrations(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	// Run schema-based migrations (CREATE TABLE IF NOT EXISTS for each resource)
	if err := runSchemaMigrations(db, resources, logger); err != nil {
		db.Close()
		return nil, fmt.Errorf("schema migrations: %w", err)
	}

	store, err := NewStore(db, resources)
	if err != nil {
		db.Close()
		return nil, err
	}

	return store, nil
}

func runFileMigrations(db *sqlx.DB) error {
	driver, err := sqlite3.WithInstance(db.DB, &sqlite3.Config{NoTxWrap: true})
	if err != nil {
		return fmt.Errorf("create migration driver: %w", err)
	}

	source, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("create migration source: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", source, "sqlite3", driver)
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("run migrations: %w", err)
	}

	return nil
}

func runSchemaMigrations(db *sqlx.DB, resources []Resource, logger *slog.Logger) error {
	for _, res := range resources {
		sql := res.GenerateCreateSQL()
		logger.Debug("ensuring table", "resource", res.Name)
		if _, err := db.Exec(sql); err != nil {
			return fmt.Errorf("create table %s: %w", res.Name, err)
		}
	}

	// Ensure ancillary tables exist (not schema-driven entities)
	ancillaryTables := []string{
		`CREATE TABLE IF NOT EXISTS usage_events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			reference_id TEXT UNIQUE NOT NULL,
			user_id INTEGER NOT NULL,
			event_type TEXT NOT NULL,
			resource_id TEXT NOT NULL DEFAULT '',
			resource_type TEXT NOT NULL DEFAULT '',
			quantity INTEGER NOT NULL DEFAULT 1,
			metadata TEXT,
			timestamp TEXT NOT NULL,
			reported_at TEXT,
			created_at TEXT NOT NULL DEFAULT (datetime('now'))
		)`,
		`CREATE INDEX IF NOT EXISTS idx_usage_events_unreported ON usage_events(reported_at) WHERE reported_at IS NULL`,
		`CREATE TABLE IF NOT EXISTS container_events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			deployment_id TEXT NOT NULL,
			type TEXT NOT NULL,
			container_name TEXT NOT NULL DEFAULT '',
			message TEXT NOT NULL DEFAULT '',
			details TEXT,
			timestamp TEXT NOT NULL DEFAULT (datetime('now'))
		)`,
		`CREATE INDEX IF NOT EXISTS idx_container_events_deployment_time ON container_events(deployment_id, timestamp DESC)`,
	}
	for _, sql := range ancillaryTables {
		if _, err := db.Exec(sql); err != nil {
			logger.Warn("ancillary table creation", "error", err)
		}
	}

	return nil
}
