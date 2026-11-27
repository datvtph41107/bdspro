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

func validateCanonicalVersion(
	ctx context.Context,
	m *migrate.Migrate,
	databaseURL string,
) error {
	version, dirty, err := m.Version()
	if errors.Is(err, migrate.ErrNilVersion) {
		return validateUnversionedDatabase(ctx, databaseURL)
	}
	if err != nil {
		return fmt.Errorf("read database migration version: %w", err)
	}
	if dirty {
		return fmt.Errorf("database migration version %d is dirty", version)
	}
	if version == 0 || version > latestSchemaVersion {
		return fmt.Errorf("unknown migration ledger version %d; provision a fresh canonical database and migrate data explicitly", version)
	}

	queryCtx, cancel := context.WithTimeout(
		ctx,
		5*time.Second,
	)
	defer cancel()

	conn, err := pgx.Connect(queryCtx, databaseURL)
	if err != nil {
		return fmt.Errorf("inspect database schema: %w", err)
	}
	defer conn.Close(context.Background())

	return validateSchema(queryCtx, conn, version)
}

func validateUnversionedDatabase(
	ctx context.Context,
	databaseURL string,
) error {
	queryCtx, cancel := context.WithTimeout(
		ctx,
		5*time.Second,
	)
	defer cancel()

	conn, err := pgx.Connect(queryCtx, databaseURL)
	if err != nil {
		return fmt.Errorf("inspect unversioned database schema: %w", err)
	}
	defer conn.Close(context.Background())

	var existing int
	err = conn.QueryRow(queryCtx, `SELECT count(*) FROM pg_catalog.pg_class c
		JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = 'public'
		  AND c.relkind IN ('r', 'p', 'v', 'm', 'S', 'f')
		  AND c.relname <> 'schema_migrations'`).Scan(&existing)
	if err != nil {
		return fmt.Errorf("inspect unversioned database objects: %w", err)
	}
	if existing > 0 {
		return fmt.Errorf("unversioned database contains %d existing public objects; refusing canonical migration; provision a fresh database and migrate data explicitly", existing)
	}
	return nil
}

func MigrateUp(
	ctx context.Context,
	sourceURL, databaseURL string,
	steps int,
) (bool, error) {
	if steps < 0 {
		return false, fmt.Errorf(
			"migration steps must not be negative",
		)
	}

	if err := ctx.Err(); err != nil {
		return false, err
	}

	m, err := migrate.New(sourceURL, databaseURL)
	if err != nil {
		return false, fmt.Errorf(
			"open migrations: %w", err,
		)
	}
	defer m.Close()

	if err := validateCanonicalVersion(
		ctx, m, databaseURL,
	); err != nil {
		return false, err
	}

	err = runMigration(ctx, m, func() error {
		if steps == 0 {
			return m.Up()
		}
		return m.Steps(steps)
	})

	if errors.Is(err, migrate.ErrNoChange) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf(
			"apply migrations: %w", err,
		)
	}

	return true, nil
}

func MigrateDown(sourceURL, databaseURL string, steps int) (bool, error) {
	if steps < 0 {
		return false, fmt.Errorf("migration steps must not be negative")
	}
	m, err := migrate.New(sourceURL, databaseURL)
	if err != nil {
		return false, fmt.Errorf("open migrations: %w", err)
	}
	defer m.Close()
	if err := validateCanonicalVersion(
		context.Background(), m, databaseURL,
	); err != nil {
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

func runMigration(
	ctx context.Context,
	m *migrate.Migrate,
	run func() error,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	done := make(chan struct{})
	defer close(done)

	go func() {
		select {
		case <-ctx.Done():
			select {
			case m.GracefulStop <- true:
			default:
			}
		case <-done:
		}
	}()

	err := run()

	if ctx.Err() != nil {
		return ctx.Err()
	}

	return err
}
