package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	httpserver "github.com/datvtph41107/bdspro/app/http/server"
	db "github.com/datvtph41107/bdspro/db/sqlc"
	"github.com/datvtph41107/bdspro/mail"
	"github.com/datvtph41107/bdspro/util"
	"github.com/datvtph41107/bdspro/worker"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx); err != nil {
		log.Fatal().Err(err).Msg("BDSPro stopped")
	}
}

func run(ctx context.Context) error {
	startedAt := time.Now()

	config, err := util.LoadConfig(".")
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	log.Logger = util.NewLogger(os.Stderr, config.LogFormat)
	if os.Getenv(gin.EnvGinMode) == "" {
		gin.SetMode(gin.ReleaseMode)
	}
	if config.LogFormat == "console" {
		fmt.Fprint(os.Stderr, "\nBDS PRO / API\n\n")
	}
	log.Info().Str("environment", config.Environment).Msg("startup.begin")

	pool, err := pgxpool.New(ctx, config.DBSource)
	if err != nil {
		return fmt.Errorf("connect PostgreSQL: %w", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("ping PostgreSQL: %w", err)
	}
	log.Info().Msg("postgres.connected")

	if err := runDBMigration(config.MigrationURL, config.DBSource); err != nil {
		return err
	}
	store := db.NewStore(pool)

	if config.RedisAddress == "" {
		return errors.New("REDIS_ADDRESS is required")
	}
	redisClient := redis.NewClient(&redis.Options{Addr: config.RedisAddress})
	defer redisClient.Close()

	redisCtx, cancelRedis := context.WithTimeout(ctx, 5*time.Second)
	err = redisClient.Ping(redisCtx).Err()
	cancelRedis()
	if err != nil {
		return fmt.Errorf("ping Redis: %w", err)
	}
	log.Info().Msg("redis.connected")

	redisOptions := asynq.RedisClientOpt{Addr: config.RedisAddress}
	distributor := worker.NewRedisTaskDistributor(redisOptions)
	defer distributor.Close()

	mailer := mail.NewSMTPSender(
		config.EmailSenderName,
		config.EmailSenderAddress,
		config.EmailSenderPassword,
		config.SMTPAddress,
		config.SMTPAuth,
	)
	processor := worker.NewRedisTaskProcessor(redisOptions, store, mailer, config.VerifyEmailURL)
	if err := processor.Start(); err != nil {
		return fmt.Errorf("start worker: %w", err)
	}
	defer processor.Shutdown()
	log.Info().Msg("task.processor.started")

	server, err := httpserver.NewServer(config, store, distributor)
	if err != nil {
		return fmt.Errorf("create HTTP server: %w", err)
	}
	if err := server.EnableAuthRateLimiting(redisClient); err != nil {
		return err
	}
	if err := server.EnableGoogleOAuth(redisClient); err != nil {
		return fmt.Errorf("enable Google OAuth: %w", err)
	}
	if err := server.EnablePhoneOTP(store); err != nil {
		return fmt.Errorf("enable phone OTP: %w", err)
	}

	return server.StartWithReady(ctx, config.HTTPServerAddress, func(net.Addr) {
		log.Info().Dur("duration", time.Since(startedAt)).Msg("startup.ready")
	})
}

func runDBMigration(migrationURL, dbSource string) error {
	m, err := migrate.New(migrationURL, dbSource)
	if err != nil {
		return fmt.Errorf("open database migrations: %w", err)
	}
	defer m.Close()

	err = m.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("apply database migrations: %w", err)
	}
	result := "applied"
	if errors.Is(err, migrate.ErrNoChange) {
		result = "no_change"
	}
	log.Info().Str("result", result).Msg("migration.checked")
	return nil
}
