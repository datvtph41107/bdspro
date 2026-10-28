package worker

import (
	"context"

	"github.com/hibiken/asynq"
	"github.com/rs/zerolog/log"
)

// TaskEnqueuer is the queue transport contract; task type and payload belong
// to the job, not to the queue transport.
type TaskEnqueuer interface {
	Enqueue(ctx context.Context, task *asynq.Task, opts ...asynq.Option) error
}

type RedisEnqueuer struct {
	client *asynq.Client
}

func NewRedisEnqueuer(options asynq.RedisClientOpt) *RedisEnqueuer {
	return &RedisEnqueuer{client: asynq.NewClient(options)}
}

func (e *RedisEnqueuer) Enqueue(ctx context.Context, task *asynq.Task, opts ...asynq.Option) error {
	info, err := e.client.EnqueueContext(ctx, task, opts...)
	if err != nil {
		return err
	}
	log.Ctx(ctx).Info().Str("type", task.Type()).Str("queue", info.Queue).
		Str("task_id", info.ID).Msg("task.enqueued")
	return nil
}

func (e *RedisEnqueuer) Close() error {
	return e.client.Close()
}

var _ TaskEnqueuer = (*RedisEnqueuer)(nil)
