package bootstrap

import (
    "context"
    "fmt"
    "os"
    "os/signal"
    "syscall"
    "time"

    db "github.com/datvtph41107/bdspro/db/sqlc"
    "github.com/datvtph41107/bdspro/util"
    "github.com/datvtph41107/bdspro/worker"
    "github.com/gin-gonic/gin"
    "github.com/go-redis/redis/v8"
    "github.com/hibiken/asynq"
    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/rs/zerolog/log"
    "golang.org/x/sync/errgroup"
)

var interruptSignals = []os.Signal{os.Interrupt, syscall.SIGTERM}

func Run() error {
	startedAt := time.Now()
	config, err := util.LoadConfig(".")
	if err != nil {
		return fmt.Errorf("cannot load config: %w", err)
	}
	log.Logger = util.NewLogger(os.Stderr, config.LogFormat)
	if os.Getenv(gin.EnvGinMode) == "" {
		gin.SetMode(gin.ReleaseMode)
	}
	if config.LogFormat == "console" {
		fmt.Fprint(os.Stderr, "\nBDS PRO / API\n\n")
	}
	log.Info().Str("environment", config.Environment).Msg("startup.begin")

	ctx, stop := signal.NotifyContext(context.Background(), interruptSignals...)
	defer stop()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	connPool, err := pgxpool.New(ctx, config.DBSource)
	if err != nil {
		return fmt.Errorf("cannot connect to db: %w", err)
	}
	defer connPool.Close()
	if err := connPool.Ping(ctx); err != nil {
		return fmt.Errorf("cannot ping db: %w", err)
	}
	log.Info().Msg("postgres.connected")

	if err := RunDBMigration(config.MigrationURL, config.DBSource); err != nil {
		return err
	}
	if config.RedisAddress == "" {
		return fmt.Errorf("REDIS_ADDRESS is required")
	}
	redisClient := redis.NewClient(&redis.Options{Addr: config.RedisAddress})
	defer redisClient.Close()
	redisCtx, cancelRedis := context.WithTimeout(ctx, 5*time.Second)
	err = redisClient.Ping(redisCtx).Err()
	cancelRedis()
	if err != nil {
		return fmt.Errorf("cannot ping redis: %w", err)
	}
	log.Info().Msg("redis.connected")

	store := db.NewStore(connPool)
	redisOpt := asynq.RedisClientOpt{Addr: config.RedisAddress}
	taskDistributor := worker.NewRedisTaskDistributor(redisOpt)
	defer taskDistributor.Close()

	waitGroup, ctx := errgroup.WithContext(ctx)

	if err := runTaskProcessor(ctx, waitGroup, config, redisOpt, store); err != nil {
		return err
	}
	waitGroup.Go(func() error {
		defer cancel() // Stop the worker if HTTP exits, including on a bind/serve error.
		return runGinServer(ctx, config, store, taskDistributor, redisClient, db.NewPhoneOTPStore(connPool), startedAt)
	})
	return waitGroup.Wait()
}

