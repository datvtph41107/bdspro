package cmd

import (
	"context"
	"fmt"

	"github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/modules/log"
	"github.com/datvtph41107/bdspro/modules/queue"
	redisconn "github.com/datvtph41107/bdspro/modules/redis"
	"github.com/datvtph41107/bdspro/services/mail"
	"github.com/datvtph41107/bdspro/services/worker"
	"github.com/hibiken/asynq"
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
	if err != nil {
		return err
	}
	store, err := db.InitStore(ctx, cfg.DBSource)
	if err != nil {
		return err
	}
	defer store.Close()
	redisClient, err := redisconn.Connect(ctx, cfg.RedisAddress)
	if err != nil {
		return err
	}
	defer redisClient.Close()

	options := asynq.RedisClientOpt{Addr: cfg.RedisAddress}
	enqueuer := queue.NewRedisEnqueuer(options)
	defer enqueuer.Close()
	mailer := mail.NewSMTPSender(cfg.EmailSenderName, cfg.EmailSenderAddress,
		cfg.EmailSenderPassword, cfg.SMTPAddress, cfg.SMTPAuth)
	processor := queue.NewRedisTaskProcessor(options)
	processor.Handle(worker.TaskSendVerifyEmail, worker.NewVerifyEmailHandler(store, mailer, cfg.VerifyEmailURL))
	processor.Handle(worker.TaskSendPasswordReset, worker.NewPasswordResetHandler(store, mailer, cfg.VerifyEmailURL))
	processor.Handle(worker.TaskSendTeamInvite, worker.NewTeamInviteHandler(store, mailer, cfg.TokenSymmetricKey))
	if err := processor.Start(); err != nil {
		return fmt.Errorf("start worker: %w", err)
	}
	defer processor.Shutdown()
	dispatchCtx, stopDispatch := context.WithCancel(ctx)
	dispatcher := worker.NewOutboxDispatcher(store, enqueuer)
	dispatcher.Start(dispatchCtx)
	defer func() { stopDispatch(); dispatcher.Wait() }()
	log.Info("Asynq worker and outbox dispatcher started")
	<-ctx.Done()
	log.Info("Worker shutdown requested")
	return nil
}
