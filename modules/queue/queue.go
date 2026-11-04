// Package queue adapts Asynq task transport and worker execution.
// PostgreSQL outbox policy and task business handlers live in services.
package queue

import (
	"context"

	"github.com/hibiken/asynq"
)

// TaskEnqueuer is the narrow existing Asynq transport contract.
// It is intentionally not a broker-independent or persistent-job interface.
type TaskEnqueuer interface {
	Enqueue(ctx context.Context, task *asynq.Task, opts ...asynq.Option) error
}
