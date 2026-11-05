package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/jackc/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// validateCanonicalVersion protects installations with the former 21-step
// migration ledger. A v21 database cannot silently masquerade as an upgraded
// instance when source control now contains only the single v1 baseline.
func validateCanonicalVersion(m *migrate.Migrate, databaseURL string) error {
	version, dirty, err := m.Version()
	if errors.Is(err, migrate.ErrNilVersion) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read database migration version: %w", err)
	}
	if dirty {
		return fmt.Errorf("database migration version %d is dirty", version)
	}
	if version != 1 {
		return fmt.Errorf("database uses migration version %d; canonical baseline v1 only supports freshly provisioned databases; export and migrate data explicitly", version)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		return fmt.Errorf("inspect database schema: %w", err)
	}
	defer conn.Close(context.Background())
	var ready bool
	err = conn.QueryRow(ctx, "SELECT to_regclass('public.oauth_clients') IS NOT NULL AND to_regclass('public.oauth2_credentials') IS NOT NULL").Scan(&ready)
	if err != nil {
		return fmt.Errorf("inspect database baseline: %w", err)
	}
	if !ready {
		return errors.New("database has a pre-baseline v1 schema; it must be migrated or re-provisioned explicitly, not reinterpreted as canonical")
	}
	return nil
}

// MigrateUp applies canonical versioned SQL migrations. steps=0 means all pending.
func MigrateUp(sourceURL, databaseURL string, steps int) (bool, error) {
	if steps < 0 {
		return false, fmt.Errorf("migration steps must not be negative")
	}
	m, err := migrate.New(sourceURL, databaseURL)
	if err != nil {
		return false, fmt.Errorf("open migrations: %w", err)
	}
	defer m.Close()
	if err := validateCanonicalVersion(m, databaseURL); err != nil {
		return false, err
	}
	if steps == 0 {
		err = m.Up()
	} else {
		err = m.Steps(steps)
	}
	if errors.Is(err, migrate.ErrNoChange) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("apply migrations: %w", err)
	}
	return true, nil
}

// MigrateDown must be restricted to approved disposable development environments.
func MigrateDown(sourceURL, databaseURL string, steps int) (bool, error) {
	if steps < 0 {
		return false, fmt.Errorf("migration steps must not be negative")
	}
	m, err := migrate.New(sourceURL, databaseURL)
	if err != nil {
		return false, fmt.Errorf("open migrations: %w", err)
	}
	defer m.Close()
	if err := validateCanonicalVersion(m, databaseURL); err != nil {
		return false, err
	}
	if steps == 0 {
		err = m.Down()
	} else {
		err = m.Steps(-steps)
	}
	if errors.Is(err, migrate.ErrNoChange) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("rollback migrations: %w", err)
	}
	return true, nil
}
