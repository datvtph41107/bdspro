package cmd

import (
    "context"
    "fmt"
    "time"

    "github.com/datvtph41107/bdspro/models/db"
    "github.com/datvtph41107/bdspro/modules/log"
    "github.com/datvtph41107/bdspro/services/mail"
    "github.com/datvtph41107/bdspro/services/worker"
    "github.com/go-redis/redis/v8"
    "github.com/hibiken/asynq"
    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/urfave/cli/v3"
)

func newWorkerCommand() *cli.Command {
    return &cli.Command{
        Name: "worker", Usage: "Process Asynq tasks and dispatch committed job outbox records",
        Before: PrepareConsoleLoggerLevel(log.INFO), Action: runWorker,
    }
}

func runWorker(ctx context.Context, command *cli.Command) error {
    if command.Args().Len() != 0 {
        return fmt.Errorf("unexpected worker argument %q", command.Args().First())
    }
    cfg, err := loadSettings(command)
    if err != nil { return err }
    pool, err := pgxpool.New(ctx, cfg.DBSource)
    if err != nil { return fmt.Errorf("connect PostgreSQL: %w", err) }
    defer pool.Close()
    if err := pool.Ping(ctx); err != nil { return fmt.Errorf("ping PostgreSQL: %w", err) }
    store := db.NewStore(pool)
    redisClient := redis.NewClient(&redis.Options{Addr:cfg.RedisAddress})
    defer redisClient.Close()
    pingCtx, cancelPing := context.WithTimeout(ctx,5*time.Second)
    err = redisClient.Ping(pingCtx).Err()
    cancelPing()
    if err != nil { return fmt.Errorf("ping Redis: %w", err) }

    options := asynq.RedisClientOpt{Addr:cfg.RedisAddress}
    enqueuer := worker.NewRedisEnqueuer(options)
    defer enqueuer.Close()
    mailer := mail.NewSMTPSender(cfg.EmailSenderName,cfg.EmailSenderAddress,
        cfg.EmailSenderPassword,cfg.SMTPAddress,cfg.SMTPAuth)
    processor := worker.NewRedisTaskProcessor(options)
    processor.Handle(worker.TaskSendVerifyEmail, worker.NewVerifyEmailHandler(store, mailer, cfg.VerifyEmailURL))
    processor.Handle(worker.TaskSendPasswordReset, worker.NewPasswordResetHandler(store, mailer, cfg.VerifyEmailURL))
    if err := processor.Start(); err != nil { return fmt.Errorf("start worker: %w", err) }
    defer processor.Shutdown()
    dispatchCtx, stopDispatch := context.WithCancel(ctx)
    dispatcher := worker.NewOutboxDispatcher(store, enqueuer)
    dispatcher.Start(dispatchCtx)
    defer func(){ stopDispatch(); dispatcher.Wait() }()
    log.Info("Asynq worker and outbox dispatcher started")
    <-ctx.Done()
    log.Info("Worker shutdown requested")
    return nil
}
