package cmd

import (
	"context"
	"fmt"

	"github.com/datvtph41107/bdspro/modules/graceful"
	"github.com/datvtph41107/bdspro/modules/queue"
	worker_service "github.com/datvtph41107/bdspro/services/worker"
	"github.com/hibiken/asynq"
)

// runWorkerProcessor runs one worker as a managed
// process activity. The caller owns application startup
// and final process shutdown.
func runWorkerProcessor(
	manager *graceful.Manager,
	redisOptions asynq.RedisClientOpt,
	config queue.Config,
	handlers []worker_service.TaskHandler,
) error {
	var runErr error

	err := manager.RunWithShutdownContext(
		func(ctx context.Context) {
			runErr = worker_service.RunProcessor(
				ctx,
				redisOptions,
				config,
				handlers,
			)
		},
	)
	if err != nil {
		return fmt.Errorf(
			"register worker activity: %w",
			err,
		)
	}

	return runErr
}
