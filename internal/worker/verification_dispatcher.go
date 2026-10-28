package worker

import (
	"context"
	"errors"
	"fmt"
	"time"

	db "github.com/datvtph41107/bdspro/internal/db/sqlc"
	"github.com/hibiken/asynq"
	"github.com/rs/zerolog/log"
)

// VerificationDispatcher polls committed registration intents and enqueues
// them to Asynq. Row locks prevent concurrent dispatchers from claiming
// the same intent. Redis failures leave the intent pending for another try.
type VerificationDispatcher struct {
	store db.Store
	distributor TaskDistributor
	done chan struct{}
}

func NewVerificationDispatcher(store db.Store, distributor TaskDistributor) *VerificationDispatcher {
	return &VerificationDispatcher{
		store: store,
		distributor: distributor,
		done: make(chan struct{}),
	}
}

func (d *VerificationDispatcher) Start(ctx context.Context) {
	go func() {
		defer close(d.done)
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			count, err := d.Dispatch(ctx)
			if err != nil && ctx.Err() == nil {
				log.Ctx(ctx).Error().Err(err).Msg("verification.outbox_dispatch_failed")
			}
			// Drain full batches immediately; sleep only when the backlog is
			// caught up or Redis is unavailable.
			if err == nil && count == 10 {
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

func (d *VerificationDispatcher) Wait() {
	<-d.done
}

// Dispatch processes a small batch. The PostgreSQL lock spans the Redis enqueue
// and queued_at update. If commit or enqueue fails, a retry may enqueue the same
// task; the stable Asynq TaskID prevents duplicate pending tasks in Redis.
func (d *VerificationDispatcher) Dispatch(ctx context.Context) (int, error) {
	count := 0
	err := d.store.ExecTx(ctx, func(q *db.Queries) error {
		pending, err := q.ListPendingVerificationOutbox(ctx, 10)
		if err != nil {
			return err
		}
		for _, record := range pending {
			taskID := fmt.Sprintf("verify-email-outbox-%d", record.ID)
			err := d.distributor.DistributeTaskSendVerifyEmail(
				ctx,
				&PayloadSendVerifyEmail{AccountID: record.AccountID},
				asynq.TaskID(taskID),
				asynq.MaxRetry(10),
				asynq.ProcessIn(10*time.Second),
				asynq.Retention(24*time.Hour),
				asynq.Queue(QueueCritical),
			)
			if err != nil && !errors.Is(err, asynq.ErrTaskIDConflict) {
				return fmt.Errorf("queue verification for account %d: %w", record.AccountID, err)
			}
			if err := q.MarkVerificationOutboxQueued(ctx, record.ID); err != nil {
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
