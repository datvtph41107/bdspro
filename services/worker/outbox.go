package worker

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/modules/queue"
	"github.com/hibiken/asynq"
)

const (
	outboxBatchSize      int32 = 10
	outboxEnqueueTimeout       = 3 * time.Second
)

// OutboxDispatcher transfers committed PostgreSQL
// job intents to Asynq.
//
// It does not execute the business tasks.
type OutboxDispatcher struct {
	enqueuer queue.TaskEnqueuer
}

func NewOutboxDispatcher(
	enqueuer queue.TaskEnqueuer,
) *OutboxDispatcher {
	if enqueuer == nil {
		panic("worker: nil task enqueuer")
	}

	return &OutboxDispatcher{
		enqueuer: enqueuer,
	}
}

// Dispatch atomically marks a batch queued only after
// Asynq accepted each task or reported a TaskID conflict.
//
// It returns the number of committed outbox rows.
func (d *OutboxDispatcher) Dispatch(
	ctx context.Context,
) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}

	count := 0

	err := db.WithTx(ctx, func(txCtx context.Context) error {
		queries, err := db.GetQueries(txCtx)
		if err != nil {
			return err
		}

		rows, err := queries.ListPendingJobOutbox(
			txCtx,
			outboxBatchSize,
		)
		if err != nil {
			return fmt.Errorf(
				"list pending outbox: %w", err,
			)
		}

		for _, row := range rows {
			if err := txCtx.Err(); err != nil {
				return err
			}

			taskID := fmt.Sprintf(
				"outbox-%d", row.ID,
			)
			if row.TaskID.Valid {
				taskID = row.TaskID.String
			}

			if taskID == "" {
				return fmt.Errorf(
					"outbox %d: empty task ID", row.ID,
				)
			}

			task := asynq.NewTask(
				row.TaskType,
				row.Payload,
			)

			options := []asynq.Option{
				asynq.TaskID(taskID),
				asynq.Queue(row.QueueName),
				asynq.MaxRetry(int(row.MaxRetry)),
				asynq.ProcessAt(row.RunAt),
			}

			enqueueCtx, cancel := context.WithTimeout(
				txCtx,
				outboxEnqueueTimeout,
			)

			enqueueErr := d.enqueuer.Enqueue(
				enqueueCtx,
				task,
				options...,
			)
			cancel()

			if enqueueErr != nil &&
				!errors.Is(
					enqueueErr,
					asynq.ErrTaskIDConflict,
				) {
				return fmt.Errorf(
					"enqueue outbox %d: %w",
					row.ID,
					enqueueErr,
				)
			}

			if err := queries.MarkJobOutboxQueued(
				txCtx,
				row.ID,
			); err != nil {
				return fmt.Errorf(
					"mark outbox %d queued: %w",
					row.ID,
					err,
				)
			}

			count++
		}

		return nil
	})

	if err != nil {
		// The PostgreSQL transaction did not commit.
		return 0, err
	}

	return count, nil
}
