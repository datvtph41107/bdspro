package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5"
)

// validateCanonicalVersion prevents reinterpreting a database created with
// another migration ledger. Each canonical migration creates exactly one table.
// In the previous ledger, a version could create, rename or drop many tables;
// a previously applied version 1..21 is therefore NOT automatically compatible.
func validateCanonicalVersion(m *migrate.Migrate, databaseURL string) error {
	version, dirty, err := m.Version()
	if errors.Is(err, migrate.ErrNilVersion) {
		return validateUnversionedDatabase(databaseURL)
	}
	if err != nil {
		return fmt.Errorf("read database migration version: %w", err)
	}
	if dirty {
		return fmt.Errorf("database migration version %d is dirty", version)
	}
	if version == 0 || version > 26 {
		return fmt.Errorf("unknown migration ledger version %d; provision a fresh canonical database and migrate data explicitly", version)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		return fmt.Errorf("inspect database schema: %w", err)
	}
	defer conn.Close(context.Background())

	var tables int
	var parties, credentials, clients, deprecated bool
	err = conn.QueryRow(ctx, `SELECT
		(SELECT count(*)::integer FROM information_schema.tables
		 WHERE table_schema = 'public' AND table_type = 'BASE TABLE'
		   AND table_name <> 'schema_migrations'),
		to_regclass('public.parties') IS NOT NULL,
		to_regclass('public.oauth2_credentials') IS NOT NULL,
		to_regclass('public.oauth_clients') IS NOT NULL,
		(to_regclass('public.google_identities') IS NOT NULL
		 OR to_regclass('public.organization_memberships') IS NOT NULL
		 OR to_regclass('public.oauth2_credentials_archive') IS NOT NULL
		 OR to_regclass('public.oauth2_identities') IS NOT NULL)`).Scan(
		&tables, &parties, &credentials, &clients, &deprecated,
	)
	if err != nil {
		return fmt.Errorf("inspect canonical schema: %w", err)
	}
	if !parties || tables != int(version) || credentials != (version >= 18) ||
		clients != (version >= 23) || deprecated {
		return fmt.Errorf("database at migration version %d does not match the canonical schema (tables=%d); use an explicit data migration; refusing reinterpretation", version, tables)
	}
	return nil
}

// validateUnversionedDatabase refuses to adopt a populated public schema as
// the canonical migration ledger. The migrate driver may create its own
// schema_migrations bookkeeping table before reaching this check.
func validateUnversionedDatabase(databaseURL string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		return fmt.Errorf("inspect unversioned database schema: %w", err)
	}
	defer conn.Close(context.Background())

	var existing int
	err = conn.QueryRow(ctx, `SELECT count(*) FROM pg_catalog.pg_class c
		JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = 'public'
		  AND c.relkind IN ('r', 'p', 'v', 'm', 'S', 'f')
		  AND c.relname <> 'schema_migrations'`).Scan(&existing)
	if err != nil {
		return fmt.Errorf("inspect unversioned database objects: %w", err)
	}
	if existing != 0 {
		return fmt.Errorf("unversioned database contains %d public objects; refusing canonical migration; provision a fresh database", existing)
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
