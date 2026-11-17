package worker

import (
	"context"
	"errors"
	"fmt"

	"github.com/datvtph41107/bdspro/modules/queue"
	"github.com/hibiken/asynq"
)

type TaskHandler struct {
	Type    string
	Handler asynq.Handler
}

var ErrNoHandlers = errors.New(
	"worker: no business task handlers registered",
)

// RunProcessor starts one Asynq processor and keeps
// the managed activity alive until shutdown completes.
func RunProcessor(
	ctx context.Context,
	redisOptions asynq.RedisClientOpt,
	config queue.Config,
	handlers []TaskHandler,
) error {
	if ctx == nil {
		return errors.New("worker: nil context")
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	if len(handlers) == 0 {
		return ErrNoHandlers
	}

	processor, err := queue.NewRedisTaskProcessorWithConfig(
		redisOptions,
		config,
	)
	if err != nil {
		return fmt.Errorf(
			"configure task processor: %w", err,
		)
	}

	registered := make(map[string]struct{}, len(handlers))

	for _, entry := range handlers {
		if entry.Type == "" || entry.Handler == nil {
			return errors.New(
				"worker: invalid task registration",
			)
		}

		if _, exists := registered[entry.Type]; exists {
			return fmt.Errorf(
				"worker: duplicate task type %q",
				entry.Type,
			)
		}

		registered[entry.Type] = struct{}{}

		processor.Handle(
			entry.Type,
			entry.Handler,
		)
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	if err := processor.Start(); err != nil {
		return fmt.Errorf(
			"start task processor: %w", err,
		)
	}

	// Start returns after launching the Asynq workers.
	// The managed activity must remain registered.
	<-ctx.Done()

	// Phase 1: stop accepting new tasks.
	processor.Stop()

	// Phase 2: drain in-flight tasks and release
	// Asynq's internal goroutines/connections.
	processor.Shutdown()

	// Only now may the lifecycle registration end.
	return nil
}
