package cmd

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/modules/log"
	redisconn "github.com/datvtph41107/bdspro/modules/redis"
	"github.com/datvtph41107/bdspro/modules/setting"
	"github.com/datvtph41107/bdspro/routers"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/urfave/cli/v3"
)

func newWebCommand() *cli.Command {
	return &cli.Command{
		Name: "web", Aliases: []string{"serve"},
		Usage:  "Start the BDSPro HTTP API",
		Before: PrepareConsoleLoggerLevel(log.INFO),
		Action: runWeb,
	}
}

func runWeb(ctx context.Context, command *cli.Command) error {
	if name, ok := isValidDefaultSubCommand(command); !ok {
		return fmt.Errorf("unknown command: %s", name)
	}
	startedAt := time.Now()
	cfg, err := loadSettings(command)
	if err != nil {
		return err
	}
	log.Info("BDSPro version: %s%s", setting.AppVer, setting.AppBuiltWith)
	log.Info("HTTP runtime starting in %s mode", cfg.Environment)

	pool, err := db.Connect(ctx, cfg.DBSource)
	if err != nil { return err }
	defer pool.Close()
	log.Info("PostgreSQL connected")

	if cfg.Environment == "development" {
		if err := runDBMigration(cfg.MigrationURL, cfg.DBSource); err != nil {
			return err
		}
	} else {
		log.Info("Database migrations are managed externally")
	}
	store := db.NewStore(pool)

	redisClient, err := redisconn.Connect(ctx, cfg.RedisAddress)
	if err != nil { return err }
	defer redisClient.Close()
	log.Info("Redis connected")

	server, err := routers.InitWebInstalled(cfg, store, redisClient)
	if err != nil {
		return fmt.Errorf("initialize HTTP routes: %w", err)
	}
	return server.StartWithReady(ctx, cfg.HTTPServerAddress, func(_ net.Addr) {
		log.Info("HTTP server ready in %s", time.Since(startedAt))
	})
}

func runDBMigration(url, source string) error {
	m, err := migrate.New(url, source)
	if err != nil {
		return fmt.Errorf("open database migrations: %w", err)
	}
	defer m.Close()
	err = m.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("apply database migrations: %w", err)
	}
	if errors.Is(err, migrate.ErrNoChange) {
		log.Info("Migration status: no_change")
	} else {
		log.Info("Migration status: applied")
	}
	return nil
}
