package sqlite3

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"log/slog"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func NewDB(ctx context.Context, dsn string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open sql db: %w", err)
	}

	err = db.PingContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to ping sql db: %w", err)
	}

	return db, nil
}

func getMigrateInstance(db *sql.DB) (*migrate.Migrate, error) {
	driver, err := sqlite.WithInstance(db, &sqlite.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to create sqlite driver: %w", err)
	}

	d, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return nil, fmt.Errorf("failed to create iofs driver: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", d, "sqlite", driver)
	if err != nil {
		return nil, fmt.Errorf("failed to create migrate instance: %w", err)
	}

	return m, nil
}

func MigrateUp(ctx context.Context, db *sql.DB) error {
	m, err := getMigrateInstance(db)
	if err != nil {
		return fmt.Errorf("failed to get migrate instance: %w", err)
	}

	err = m.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to apply migration: %w", err)
	}

	version, dirty, err := m.Version()
	if err != nil {
		return fmt.Errorf("failed to get current active migration version: %w", err)
	}

	slog.InfoContext(ctx, "migration applied successfully", "version", version, "dirty", dirty)

	return nil
}

func MigrateDown(db *sql.DB) error {
	m, err := getMigrateInstance(db)
	if err != nil {
		return fmt.Errorf("failed to get migrate instance: %w", err)
	}

	err = m.Down()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to apply migration down: %w", err)
	}

	return nil
}
