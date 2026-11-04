package cmd

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/modules/log"
	redisconn "github.com/datvtph41107/bdspro/modules/redis"
	"github.com/datvtph41107/bdspro/modules/setting"
	"github.com/datvtph41107/bdspro/routers"
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

	store, err := db.InitStore(ctx, cfg.DBSource)
	if err != nil {
		return err
	}
	defer store.Close()
	log.Info("PostgreSQL connected")

	if cfg.Environment == "development" {
		changed, err := db.MigrateUp(cfg.MigrationURL, cfg.DBSource, 0)
		if err != nil {
			return err
		}
		if changed {
			log.Info("Migration status: applied")
		} else {
			log.Info("Migration status: no_change")
		}
	} else {
		log.Info("Database migrations are managed externally")
	}

	redisClient, err := redisconn.Connect(ctx, cfg.RedisAddress)
	if err != nil {
		return err
	}
	defer redisClient.Close()
	log.Info("Redis connected")

	server, err := routers.InitWebInstalled(cfg, store, redisClient)
	if err != nil {
		return fmt.Errorf("initialize HTTP routes: %w", err)
	}
	return serveHTTP(ctx, server.Handler(), cfg.HTTPServerAddress, func(_ net.Addr) {
		log.Info("HTTP server ready in %s", time.Since(startedAt))
	})
}
