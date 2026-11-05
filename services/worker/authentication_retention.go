package worker

import (
	"context"
	"fmt"
	"time"

	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/modules/log"
)

const (
	authenticationRetentionWindow     = 90 * 24 * time.Hour
	authenticationRetentionInterval   = time.Hour
	authenticationRetentionBatchSize  = int32(1000)
	authenticationRetentionMaxBatches = 20
)

// AuthenticationRetention expires successful connection histories and rejected
// proof events. This is operational maintenance, never a login dependency.
// Multiple workers may run the SQLC SKIP LOCKED deletes concurrently.
type AuthenticationRetention struct {
	store interface {
		DeleteExpiredAccountConnectionHistories(context.Context, db.DeleteExpiredAccountConnectionHistoriesParams) (int64, error)
		DeleteExpiredAuthenticationFailures(context.Context, db.DeleteExpiredAuthenticationFailuresParams) (int64, error)
	}
	done chan struct{}
}

func NewAuthenticationRetention(store interface {
	DeleteExpiredAccountConnectionHistories(context.Context, db.DeleteExpiredAccountConnectionHistoriesParams) (int64, error)
	DeleteExpiredAuthenticationFailures(context.Context, db.DeleteExpiredAuthenticationFailuresParams) (int64, error)
}) *AuthenticationRetention {
	return &AuthenticationRetention{store: store, done: make(chan struct{})}
}

// Sweep applies the same cutoff independently to both tables, at most
// 20,000 rows per table per pass. A partial failure is retried at the next
// scheduled sweep and never rolls back an earlier committed deletion.
func (w *AuthenticationRetention) Sweep(ctx context.Context, now time.Time) (int64, error) {
	cutoff := now.UTC().Add(-authenticationRetentionWindow)
	var deleted int64
	for i := 0; i < authenticationRetentionMaxBatches; i++ {
		if err := ctx.Err(); err != nil {
			return deleted, err
		}
		n, err := w.store.DeleteExpiredAccountConnectionHistories(ctx, db.DeleteExpiredAccountConnectionHistoriesParams{
			Cutoff: cutoff,
			Take:   authenticationRetentionBatchSize,
		})
		if err != nil {
			return deleted, err
		}
		deleted += n
		if n < int64(authenticationRetentionBatchSize) {
			break
		}
	}
	for i := 0; i < authenticationRetentionMaxBatches; i++ {
		if err := ctx.Err(); err != nil {
			return deleted, err
		}
		n, err := w.store.DeleteExpiredAuthenticationFailures(ctx, db.DeleteExpiredAuthenticationFailuresParams{
			Cutoff: cutoff,
			Take:   authenticationRetentionBatchSize,
		})
		if err != nil {
			return deleted, err
		}
		deleted += n
		if n < int64(authenticationRetentionBatchSize) {
			break
		}
	}
	return deleted, nil
}

func (w *AuthenticationRetention) Start(ctx context.Context) {
	go func() {
		defer close(w.done)
		ticker := time.NewTicker(authenticationRetentionInterval)
		defer ticker.Stop()
		for {
			sweepCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
			deleted, err := w.Sweep(sweepCtx, time.Now())
			cancel()
			if err != nil && ctx.Err() == nil {
				log.Ctx(ctx).Error().Str("error_type", fmt.Sprintf("%T", err)).Int64("deleted", deleted).Msg("auth.retention_failed")
			} else if err == nil && deleted > 0 {
				log.Ctx(ctx).Info().Int64("deleted", deleted).Msg("auth.histories_retained")
			}
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}

func (w *AuthenticationRetention) Wait() { <-w.done }
