package worker

import (
	"context"
	"errors"
	"fmt"
	"time"

	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/modules/log"
)

const (
	loginRetentionInterval   = time.Hour
	loginRetentionBatchSize  = int32(1000)
	loginRetentionMaxBatches = 20
)

// LoginRetention expires successful connection histories and rejected
// audit events. This is operational maintenance, never a login dependency.
// Multiple workers may run the SQLC SKIP LOCKED deletes concurrently.
type LoginRetention struct {
	retention time.Duration
	done      chan struct{}
}

func NewLoginRetention(retentionDays int) *LoginRetention {
	return &LoginRetention{
		retention: time.Duration(retentionDays) * 24 * time.Hour,
		done:      make(chan struct{}),
	}
}

// Sweep applies the same cutoff independently to both tables, at most
// 20,000 rows per table per pass. Even if connection cleanup fails, the
// rejected-proof cleanup is still attempted. Partial failures are retried on
// the next scheduled sweep and never roll back committed deletions.
func (w *LoginRetention) Sweep(ctx context.Context, now time.Time) (int64, error) {
	q, err := db.GetQueries(ctx)
	if err != nil {
		return 0, err
	}
	cutoff := now.UTC().Add(-w.retention)
	var deleted int64
	var connectionErr error
	for i := 0; i < loginRetentionMaxBatches; i++ {
		if err := ctx.Err(); err != nil {
			return deleted, err
		}
		n, err := q.DeleteExpiredAccountConnectionHistories(ctx, db.DeleteExpiredAccountConnectionHistoriesParams{
			Cutoff: cutoff,
			Take:   loginRetentionBatchSize,
		})
		if err != nil {
			connectionErr = err
			break
		}
		deleted += n
		if n < int64(loginRetentionBatchSize) {
			break
		}
	}
	for i := 0; i < loginRetentionMaxBatches; i++ {
		if err := ctx.Err(); err != nil {
			return deleted, errors.Join(connectionErr, err)
		}
		n, err := q.DeleteOldAuditEvents(ctx, db.DeleteOldAuditEventsParams{
			Cutoff: cutoff,
			Take:   loginRetentionBatchSize,
		})
		if err != nil {
			return deleted, errors.Join(connectionErr, err)
		}
		deleted += n
		if n < int64(loginRetentionBatchSize) {
			break
		}
	}
	return deleted, connectionErr
}

func (w *LoginRetention) Start(ctx context.Context) {
	go func() {
		defer close(w.done)
		ticker := time.NewTicker(loginRetentionInterval)
		defer ticker.Stop()
		for {
			sweepCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
			deleted, err := w.Sweep(sweepCtx, time.Now())
			cancel()
			if err != nil && ctx.Err() == nil {
				log.Ctx(ctx).Error().Str("error_type", fmt.Sprintf("%T", err)).Int64("deleted", deleted).Msg("audit.retention_failed")
			} else if err == nil && deleted > 0 {
				log.Ctx(ctx).Info().Int64("deleted", deleted).Msg("audit.histories_pruned")
			}
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}

func (w *LoginRetention) Wait() { <-w.done }
