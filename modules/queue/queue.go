// Package queue owns durable Asynq task transport and worker lifecycle.
// BDSPro's committed SQL outbox must not silently fall back to an in-memory queue.
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
