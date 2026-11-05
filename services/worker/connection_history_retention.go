package worker

import (
	"context"
	"fmt"
	"time"

	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/modules/log"
)

const (
	connectionHistoryRetention  = 90 * 24 * time.Hour
	connectionHistoryInterval   = time.Hour
	connectionHistoryBatchSize  = int32(1000)
	connectionHistoryMaxBatches = 20
)

// ConnectionHistoryRetention is an operational worker, not an authentication
// request dependency. Multiple workers may execute it safely: SQL rows are
// selected with SKIP LOCKED, and deletion is idempotent.
type ConnectionHistoryRetention struct {
	store interface {
		DeleteExpiredAccountConnectionHistories(context.Context, db.DeleteExpiredAccountConnectionHistoriesParams) (int64, error)
	}
	done chan struct{}
}

func NewConnectionHistoryRetention(store interface {
	DeleteExpiredAccountConnectionHistories(context.Context, db.DeleteExpiredAccountConnectionHistoriesParams) (int64, error)
}) *ConnectionHistoryRetention {
	return &ConnectionHistoryRetention{store: store, done: make(chan struct{})}
}

// Sweep deletes at most 20,000 old records per pass to bound database work;
// further records are processed on the next hourly pass. Caller supplies the
// clock for reproducible cutoff tests.
func (w *ConnectionHistoryRetention) Sweep(ctx context.Context, now time.Time) (int64, error) {
	cutoff := now.UTC().Add(-connectionHistoryRetention)
	var deleted int64
	for i := 0; i < connectionHistoryMaxBatches; i++ {
		if err := ctx.Err(); err != nil {
			return deleted, err
		}
		n, err := w.store.DeleteExpiredAccountConnectionHistories(ctx, db.DeleteExpiredAccountConnectionHistoriesParams{
			Cutoff: cutoff,
			Take:   connectionHistoryBatchSize,
		})
		if err != nil {
			return deleted, err
		}
		deleted += n
		if n < int64(connectionHistoryBatchSize) {
			break
		}
	}
	return deleted, nil
}

func (w *ConnectionHistoryRetention) Start(ctx context.Context) {
	go func() {
		defer close(w.done)
		ticker := time.NewTicker(connectionHistoryInterval)
		defer ticker.Stop()
		for {
			sweepCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
			deleted, err := w.Sweep(sweepCtx, time.Now())
			cancel()
			if err != nil && ctx.Err() == nil {
				log.Ctx(ctx).Error().Str("error_type", fmt.Sprintf("%T", err)).Msg("auth.connection_history_retention_failed")
			} else if err == nil && deleted > 0 {
				log.Ctx(ctx).Info().Int64("deleted", deleted).Msg("auth.connection_history_retained")
			}
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}

func (w *ConnectionHistoryRetention) Wait() { <-w.done }
