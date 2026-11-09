package cmd

import (
	"context"
	"fmt"

	"github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/modules/conn"
	"github.com/datvtph41107/bdspro/modules/log"
	"github.com/datvtph41107/bdspro/modules/queue"
	"github.com/datvtph41107/bdspro/services/mail"
	"github.com/datvtph41107/bdspro/services/worker"
	"github.com/hibiken/asynq"
	"github.com/urfave/cli/v3"
)

func newWorkerCommand() *cli.Command {
	return &cli.Command{
		Name: "worker", Usage: "Process Asynq tasks, dispatch outbox records and expire login/audit history",
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
	redisManager := conn.NewRedisManager()
	defer redisManager.Close() // manager retains every client it creates
	_, err = redisManager.Client(ctx, cfg.RedisAddress)
	if err != nil {
		return err
	}

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
	queueManager := queue.NewManager()
	if err := queueManager.Register("tasks", processor); err != nil {
		processor.Shutdown()
		return err
	}
	defer queueManager.Shutdown()
	dispatchCtx, stopDispatch := context.WithCancel(ctx)
	dispatcher := worker.NewOutboxDispatcher(store, enqueuer)
	dispatcher.Start(dispatchCtx)
	retention := worker.NewLoginRetention(store, cfg.AuditRetentionDays)
	retention.Start(dispatchCtx)
	defer func() { stopDispatch(); retention.Wait(); dispatcher.Wait() }()
	log.Info("Asynq worker, outbox dispatcher and login and audit retention started")
	<-ctx.Done()
	log.Info("Worker shutdown requested")
	return nil
}
