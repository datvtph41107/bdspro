package cmd

import (
	"context"

	"github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/modules/conn"
	"github.com/datvtph41107/bdspro/modules/graceful"
	"github.com/datvtph41107/bdspro/modules/log"
	"github.com/datvtph41107/bdspro/modules/process"
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
	setting.LoadDBSetting()
	setting.LoadRedisSetting()
	if err := setting.LoadTokenSigningKey(); err != nil {
		return err
	}
	if err := setting.LoadAuditRetentionDays(); err != nil {
		return err
	}
	if err := setting.LoadQueueSetting(); err != nil {
		return err
	}
	if err := setting.LoadMailerSetting(); err != nil {
		return err
	}

	graceful.InitManagerWithTimeout(ctx, setting.Common.Graceful.HammerTimeout)
	lifecycle := graceful.GetManager()
	lifecycle.RegisterServer()
	err := func() error {
		defer lifecycle.ServerDone()
		workerCtx, _, finished := process.GetManager().AddTypedContext(
			lifecycle.HammerContext(), "Worker: Asynq and outbox", process.SystemProcessType, true,
		)
		defer finished()
		return runWorkerRuntime(workerCtx, lifecycle.ShutdownContext())
	}()
	lifecycle.DoGracefulShutdown()
	<-lifecycle.Done()
	return err
}

// runWorkerRuntime owns resources. All defers finish before the worker is
// marked drained; database and Redis close only after the jobs stop.
func runWorkerRuntime(ctx, shutdownCtx context.Context) error {
	store, err := db.InitStore(ctx, setting.Database.DBSource)
	if err != nil {
		return err
	}
	defer store.Close()

	redisManager := conn.NewRedisManager()
	defer redisManager.Close()
	if _, err = redisManager.Client(ctx, setting.Redis.Address); err != nil {
		return err
	}

	options := asynq.RedisClientOpt{Addr: setting.Redis.Address}
	enqueuer := queue.NewRedisEnqueuer(options)
	defer enqueuer.Close()

	mailer := mail.NewSMTPSender(setting.Mailer.EmailSenderName, setting.Mailer.EmailSenderAddress,
		setting.Mailer.EmailSenderPassword, setting.Mailer.SMTPAddress, setting.Mailer.SMTPAuth)
	processor, err := queue.NewRedisTaskProcessorWithConfig(options, queue.Config{
		Queues: map[string]int{
			queue.QueueCritical: setting.Queue.CriticalWeight,
			queue.QueueDefault:  setting.Queue.DefaultWeight,
		},
		Concurrency: setting.Queue.Concurrency,
	})
	if err != nil {
		return err
	}
	processor.Handle(worker.TaskSendVerifyEmail, worker.NewVerifyEmailHandler(store, mailer, setting.Mailer.VerifyEmailURL))
	processor.Handle(worker.TaskSendPasswordReset, worker.NewPasswordResetHandler(store, mailer, setting.Mailer.VerifyEmailURL))
	processor.Handle(worker.TaskSendTeamInvite, worker.NewTeamInviteHandler(store, mailer, setting.Security.TokenSymmetricKey))
	if err = processor.Start(); err != nil {
		return err
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
	retention := worker.NewLoginRetention(store, setting.Audit.RetentionDays)
	retention.Start(dispatchCtx)
	defer func() { stopDispatch(); retention.Wait(); dispatcher.Wait() }()

	log.Info("Asynq worker, outbox dispatcher and retention started")
	<-shutdownCtx.Done()
	log.Info("Worker shutdown requested")
	return nil
}
