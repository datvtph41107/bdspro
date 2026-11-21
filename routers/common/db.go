package common

import (
	"context"
	"fmt"

	"github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/modules/log"
	"github.com/datvtph41107/bdspro/modules/setting"
)

// InitDBEngine prepares the database for the installed
// Web application. Connection ownership belongs to db.
func InitDBEngine(ctx context.Context, runMode string, cfg setting.DatabaseSettings) (err error) {
	if err := ctx.Err(); err != nil {
		return err
	}

	log.Info("Beginning PostgreSQL initialization")

	if err := db.InitEngine(ctx, cfg.DBSource); err != nil {
		return fmt.Errorf(
			"initialize PostgreSQL engine: %w", err,
		)
	}

	// Once the engine is initialized, every subsequent
	// startup failure must release it.
	defer func() {
		if err != nil {
			db.CloseEngine()
		}
	}()

	if err := ctx.Err(); err != nil {
		return err
	}

	if runMode != "development" {
		log.Info("Database migrations are managed externally")
		return nil
	}

	changed, err := db.MigrateUp(
		ctx,
		cfg.MigrationURL,
		cfg.DBSource,
		0,
	)
	if err != nil {
		return fmt.Errorf(
			"migrate PostgreSQL schema: %w", err,
		)
	}

	if changed {
		log.Info("Migration status: applied")
	} else {
		log.Info("Migration status: no_change")
	}

	return ctx.Err()
}
