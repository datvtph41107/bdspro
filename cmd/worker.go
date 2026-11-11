package cmd

import (
	"context"
	"fmt"

	"github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/modules/conn"
	"github.com/datvtph41107/bdspro/modules/log"
	"github.com/datvtph41107/bdspro/modules/queue"
	"github.com/datvtph41107/bdspro/modules/setting"
	"github.com/datvtph41107/bdspro/services/mail"
	"github.com/datvtph41107/bdspro/services/worker"
	"github.com/hibiken/asynq"
	"github.com/urfave/cli/v3"
)

func newWorkerCommand() *cli.Command {
	return &cli.Command{
		Name: "worker", Usage: "Process Asynq tasks, outbox, and retention jobs",
		Before: rejectArguments("worker"), Action: runWorker,
	}
}

func runWorker(ctx context.Context, _ *cli.Command) error {
	cfg, err := setting.LoadWorkerSettings()
	if err != nil {
		return fmt.Errorf("load worker settings: %w", err)
	}
	store, err := db.InitStore(ctx, cfg.DBSource)
	if err != nil {
		return err
	}
	defer store.Close()

	redisManager := conn.NewRedisManager()
	defer redisManager.Close()
	if _, err = redisManager.Client(ctx, cfg.RedisAddress); err != nil {
		return err
	}

	options := asynq.RedisClientOpt{Addr: cfg.RedisAddress}
	enqueuer := queue.NewRedisEnqueuer(options)
	defer enqueuer.Close()

	mailer := mail.NewSMTPSender(cfg.EmailSenderName, cfg.EmailSenderAddress,
		cfg.EmailSenderPassword, cfg.SMTPAddress, cfg.SMTPAuth)
	processor, err := queue.NewRedisTaskProcessorWithConfig(options, queue.Config{
		Queues: map[string]int{
			queue.QueueCritical: cfg.Queue.CriticalWeight,
			queue.QueueDefault:  cfg.Queue.DefaultWeight,
		},
		Concurrency: cfg.Queue.Concurrency,
	})
	if err != nil {
		return fmt.Errorf("configure Asynq worker: %w", err)
	}

	processor.Handle(worker.TaskSendVerifyEmail, worker.NewVerifyEmailHandler(store, mailer, cfg.VerifyEmailURL))
	processor.Handle(worker.TaskSendPasswordReset, worker.NewPasswordResetHandler(store, mailer, cfg.VerifyEmailURL))
	processor.Handle(worker.TaskSendTeamInvite, worker.NewTeamInviteHandler(store, mailer, cfg.TokenSymmetricKey))
	if err = processor.Start(); err != nil {
		return fmt.Errorf("start worker: %w", err)
	}

	manager := queue.NewManager()
	if err = manager.Register("tasks", processor); err != nil {
		processor.Shutdown()
		return err
	}
	defer manager.Shutdown()

	dispatchCtx, stopDispatch := context.WithCancel(ctx)
	dispatcher := worker.NewOutboxDispatcher(store, enqueuer)
	dispatcher.Start(dispatchCtx)
	retention := worker.NewLoginRetention(store, cfg.AuditRetentionDays)
	retention.Start(dispatchCtx)
	defer func() {
		stopDispatch()
		retention.Wait()
		dispatcher.Wait()
	}()

	log.Info("Asynq worker, outbox dispatcher and retention started")
	<-ctx.Done()
	log.Info("Worker shutdown requested")
	return nil
}
