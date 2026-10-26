package bootstrap

import (
    "errors"
    "fmt"

    "github.com/golang-migrate/migrate/v4"
    _ "github.com/golang-migrate/migrate/v4/database/postgres"
    _ "github.com/golang-migrate/migrate/v4/source/file"
    "github.com/rs/zerolog/log"
)

func RunDBMigration(migrationURL, dbSource string) error {
	migration, err := migrate.New(migrationURL, dbSource)
	if err != nil {
		return fmt.Errorf("cannot create migrate instance: %w", err)
	}
	defer migration.Close()

	err = migration.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("cannot apply database migrations: %w", err)
	}
	result := "applied"
	if errors.Is(err, migrate.ErrNoChange) {
		result = "no_change"
	}
	log.Info().Str("result", result).Msg("migration.checked")
	return nil
}

