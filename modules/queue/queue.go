// Package queue owns task transport, worker execution and process lifecycle.
// Redis uses Asynq. The channel backend is deliberately non-durable.
package queue

import (
	"context"

	"github.com/hibiken/asynq"
)

// TaskEnqueuer is the existing narrow contract consumed by the outbox worker.
// Asynq-specific options are intentionally visible; this is not a generic broker API.
type TaskEnqueuer interface {
	Enqueue(context.Context, *asynq.Task, ...asynq.Option) error
}
