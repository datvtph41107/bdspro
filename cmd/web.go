package cmd

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/modules/log"
	"github.com/datvtph41107/bdspro/routers"
	"github.com/datvtph41107/bdspro/services/mail"
	"github.com/go-redis/redis/v8"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
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
	log.Info("BDSPro version: %s%s", "development", "")
	log.Info("HTTP runtime starting in %s mode", cfg.Environment)

	pool, err := pgxpool.New(ctx, cfg.DBSource)
	if err != nil {
		return fmt.Errorf("connect PostgreSQL: %w", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("ping PostgreSQL: %w", err)
	}
	log.Info("PostgreSQL connected")

	if cfg.Environment == "development" {
		if err := runDBMigration(cfg.MigrationURL, cfg.DBSource); err != nil {
			return err
		}
	} else {
		log.Info("Database migrations are managed externally")
	}
	store := db.NewStore(pool)

	if cfg.RedisAddress == "" {
		return errors.New("Redis address is required")
	}
	redisClient := redis.NewClient(&redis.Options{Addr: cfg.RedisAddress})
	defer redisClient.Close()
	checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	err = redisClient.Ping(checkCtx).Err()
	cancel()
	if err != nil {
		return fmt.Errorf("ping Redis: %w", err)
	}
	log.Info("Redis connected")

	mailer := mail.NewSMTPSender(cfg.EmailSenderName, cfg.EmailSenderAddress,
		cfg.EmailSenderPassword, cfg.SMTPAddress, cfg.SMTPAuth)
	server, err := routers.InitWebInstalled(cfg, store, redisClient, mailer)
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
