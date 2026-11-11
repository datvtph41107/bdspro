package cmd

import (
	"context"
	"errors"

	"github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/modules/log"
	"github.com/datvtph41107/bdspro/modules/setting"
	"github.com/urfave/cli/v3"
)

func newMigrateCommand() *cli.Command {
	return &cli.Command{
		Name: "migrate", Usage: "Apply versioned SQL migrations",
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

func runMigrate(_ context.Context, command *cli.Command) error {
	setting.LoadDBSetting()
	steps := command.Int("steps")
	if command.Bool("down") {
		if setting.Common.RunMode != "development" {
			return errors.New("schema rollback is restricted to development")
		}
		changed, err := db.MigrateDown(setting.Database.MigrationURL, setting.Database.DBSource, steps)
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
	changed, err := db.MigrateUp(setting.Database.MigrationURL, setting.Database.DBSource, steps)
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
