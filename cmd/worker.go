package cmd

import (
	"context"
	"errors"

	"github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/modules/conn"
	"github.com/datvtph41107/bdspro/modules/graceful"
	"github.com/datvtph41107/bdspro/modules/log"
	"github.com/datvtph41107/bdspro/modules/queue"
	"github.com/datvtph41107/bdspro/modules/setting"
	"github.com/datvtph41107/bdspro/modules/startup"
	worker_service "github.com/datvtph41107/bdspro/services/worker"
	"github.com/hibiken/asynq"
	"github.com/urfave/cli/v3"
)

func newWorkerCommand() *cli.Command {
	return &cli.Command{
		Name:   "worker",
		Usage:  "Run BDSPro background workers",
		Before: rejectArguments("worker"),
		Action: runWorker,
	}
}

func runWorker(
	ctx context.Context,
	_ *cli.Command,
) error {
	managerCtx, cancel := context.WithCancel(ctx)
	graceful.InitManager(managerCtx)
	defer cancel()

	return serveWorker()
}

// serveWorker composes dependencies and runs one worker
// until the process enters graceful shutdown.
func serveWorker() error {
	manager := graceful.GetManager()

	// Protect the entire startup from early termination.
	if err := manager.RegisterServer(); err != nil {
		<-manager.Done()
		if errors.Is(err, graceful.ErrShuttingDown) {
			return nil
		}
		return err
	}

	runErr := func() error {
		defer manager.ServerDone()

		if err := manager.ShutdownContext().Err(); err != nil {
			return err
		}

		// Config is validated before opening DB/Redis; a failed stage
		// releases all previously initialized resources in reverse order.
		// ShutdownContext cancels an in-flight worker bootstrap promptly.
		if err := startup.Run(manager.ShutdownContext(),
			startup.Step{
				Name: "worker settings",
				Init: func(context.Context) error {
					return setting.LoadWorkerSettings()
				},
			},
			startup.Step{
				Name: "PostgreSQL",
				Init: db.InitEngine,
				Rollback: db.CloseEngine,
			},
			startup.Step{
				Name: "Redis",
				Init: func(ctx context.Context) error {
					_, err := conn.InitRedis(ctx)
					return err
				},
				Rollback: conn.CloseRedis,
			},
		); err != nil {
			return err
		}

		if err := manager.ShutdownContext().Err(); err != nil {
			return err
		}

		deps, err := newWorkerHandlerDependencies()
		if err != nil {
			return err
		}

		handlers, err := worker_service.TaskHandlers(deps)
		if err != nil {
			return err
		}

		if len(handlers) == 0 {
			return worker_service.ErrNoHandlers
		}

		config := queue.Config{
			Concurrency: setting.Queue.Concurrency,
			Queues: map[string]int{
				queue.QueueCritical: setting.Queue.CriticalWeight,
				queue.QueueDefault:  setting.Queue.DefaultWeight,
			},
			ShutdownTimeout: setting.Queue.ShutdownTimeout,
		}

		redisOptions := asynq.RedisClientOpt{
			Addr: setting.Redis.Address,
		}

		log.Info("Starting BDSPro worker")

		var workerErr error

		// The activity remains registered until
		// RunProcessor has stopped the Asynq server.
		registrationErr := manager.RunWithShutdownContext(
			func(runCtx context.Context) {
				workerErr = worker_service.RunProcessor(
					runCtx,
					redisOptions,
					config,
					handlers,
				)
			},
		)

		if registrationErr != nil {
			return registrationErr
		}

		return workerErr
	}()

	// All code paths enter orderly process termination.
	manager.DoGracefulShutdown()
	<-manager.Done()

	if errors.Is(runErr, context.Canceled) &&
		manager.ShutdownContext().Err() != nil {
		return nil
	}

	return runErr
}
