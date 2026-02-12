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

	// If DB was managed by old migrations (v2-v11), force to our single
	// migration version. Schema is now engine-driven (CREATE TABLE IF NOT EXISTS).
	version, dirty, err := m.Version()
	if err == nil && !dirty && version > 1 {
		if err := m.Force(1); err != nil {
			return fmt.Errorf("force migration version: %w", err)
		}
		return nil
	}

	// Fresh DB or at version 1: run normally
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

	// Add columns that CREATE TABLE IF NOT EXISTS won't add to existing tables.
	// Older production databases may be missing standard columns (created_at, updated_at)
	// that the engine now requires.
	var alterStatements []string

	// Ensure every engine-managed table has created_at and updated_at.
	// SQLite ALTER TABLE ADD COLUMN requires constant defaults, so use empty string.
	// The engine's Create() always sets these explicitly.
	for _, res := range resources {
		alterStatements = append(alterStatements,
			fmt.Sprintf(`ALTER TABLE %s ADD COLUMN created_at TEXT DEFAULT ''`, res.Name),
			fmt.Sprintf(`ALTER TABLE %s ADD COLUMN updated_at TEXT DEFAULT ''`, res.Name),
		)
	}

	// Entity-specific migrations
	alterStatements = append(alterStatements,
		`ALTER TABLE nodes ADD COLUMN public INTEGER DEFAULT 0`,
		`ALTER TABLE ssh_keys RENAME COLUMN private_key_encrypted TO private_key`,
		`ALTER TABLE cloud_credentials RENAME COLUMN credentials_encrypted TO credentials`,
	)

	for _, sql := range alterStatements {
		if _, err := db.Exec(sql); err != nil {
			// Ignore "duplicate column" / "no such column" errors — column may already exist
			logger.Debug("alter table (may already exist)", "sql", sql, "error", err)
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
			reference_id TEXT UNIQUE NOT NULL,
			deployment_id INTEGER NOT NULL,
			type TEXT NOT NULL,
			container TEXT NOT NULL DEFAULT '',
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
