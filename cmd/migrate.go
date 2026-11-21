package cmd

import (
	"context"
	"errors"
	"os"

	"github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/modules/log"
	"github.com/datvtph41107/bdspro/modules/setting"
	"github.com/urfave/cli/v3"
)

func newMigrateCommand() *cli.Command {
	return &cli.Command{
		Name:  "migrate",
		Usage: "Apply versioned SQL migrations",
		Flags: []cli.Flag{
			&cli.IntFlag{Name: "steps", Usage: "Number of versions (0 means all)"},
			&cli.BoolFlag{Name: "down", Usage: "Rollback migrations (development only)"},
		},
		Before: func(ctx context.Context, c *cli.Command) (context.Context, error) {
			next, err := rejectArguments("migrate")(ctx, c)
			if err != nil {
				return next, err
			}
			if c.Int("steps") < 0 {
				return ctx, errors.New("--steps cannot be negative")
			}
			return ctx, nil
		},
		Action: runMigrate,
	}
}

// runMigrate nhận DB source và migration URL từ canonical setting.
func runMigrate(ctx context.Context, command *cli.Command) error {
	cfg, err := setting.LoadMigrateConfig(configPathOptions(command), os.Environ())
	if err != nil {
		return err
	}
	if err := setting.InitLoggers(cfg.Common.Log); err != nil {
		return err
	}
	log.Info("Using configuration file: %s", cfg.Paths.ConfigFile)
	steps := command.Int("steps")
	if command.Bool("down") {
		if cfg.Common.RunMode != "development" {
			return errors.New("schema rollback is restricted to development")
		}
		changed, err := db.MigrateDown(cfg.Database.MigrationURL, cfg.Database.DBSource, steps)
		if err != nil {
			return err
		}
		if changed {
			log.Info("Migration status: rolled_back")
		} else {
			log.Info("Migration status: no_change")
		}
		return nil
	}

	changed, err := db.MigrateUp(ctx, cfg.Database.MigrationURL, cfg.Database.DBSource, steps)
	if err != nil {
		return err
	}
	if changed {
		log.Info("Migration status: applied")
	} else {
		log.Info("Migration status: no_change")
	}
	return nil
}
