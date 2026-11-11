package worker

import (
	"context"
	"errors"
	"fmt"
	"time"

	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/modules/log"
	"github.com/datvtph41107/bdspro/modules/queue"
	"github.com/hibiken/asynq"
)

const outboxBatchSize int32 = 10

// OutboxDispatcher delivers committed job intents to Asynq.
// It does not know which business operation a task represents.
type OutboxDispatcher struct {
	enqueuer queue.TaskEnqueuer
	done     chan struct{}
}

func NewOutboxDispatcher(enqueuer queue.TaskEnqueuer) *OutboxDispatcher {
	return &OutboxDispatcher{enqueuer: enqueuer, done: make(chan struct{})}
}

func (d *OutboxDispatcher) Start(ctx context.Context) {
	go func() {
		defer close(d.done)
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			n, err := d.Dispatch(ctx)
			if err != nil && ctx.Err() == nil {
				log.Ctx(ctx).Error().
					Str("error_type", fmt.Sprintf("%T", err)).
					Msg("queue.outbox_dispatch_failed")
			}
			if err == nil && n == int(outboxBatchSize) {
				continue
			}
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}

func (d *OutboxDispatcher) Wait() { <-d.done }

// Dispatch claims pending records with PostgreSQL row locks. Redis enqueue
// and PostgreSQL commit are not atomic; task IDs reduce, but cannot eliminate,
// duplicates. Jobs must tolerate at-least-once execution.
func (d *OutboxDispatcher) Dispatch(ctx context.Context) (int, error) {
	count := 0
	err := db.WithTx(ctx, func(txCtx context.Context) error {
		q, err := db.GetQueries(txCtx)
		if err != nil {
			return err
		}
		pending, err := q.ListPendingJobOutbox(txCtx, outboxBatchSize)
		if err != nil {
			return err
		}
		for _, record := range pending {
			id := fmt.Sprintf("job-outbox-%d", record.ID)
			if record.TaskID.Valid {
				id = record.TaskID.String
			}
			task := asynq.NewTask(record.TaskType, record.Payload)
			opts := []asynq.Option{
				asynq.TaskID(id),
				asynq.Queue(record.QueueName),
				asynq.MaxRetry(int(record.MaxRetry)),
				asynq.ProcessAt(record.RunAt),
				asynq.Retention(24 * time.Hour),
			}
			enqueueCtx, cancel := context.WithTimeout(txCtx, 3*time.Second)
			err := d.enqueuer.Enqueue(enqueueCtx, task, opts...)
			cancel()
			if err != nil && !errors.Is(err, asynq.ErrTaskIDConflict) {
				return fmt.Errorf("enqueue outbox record %d: %w", record.ID, err)
			}
			if err := q.MarkJobOutboxQueued(txCtx, record.ID); err != nil {
				return err
			}
			count++
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return count, nil
}
