// Package audit coordinates bounded audit maintenance jobs.
package audit

import (
	"context"
	"errors"
	"fmt"
	"time"

	auditmodel "github.com/datvtph41107/bdspro/models/audit"
	"github.com/datvtph41107/bdspro/modules/log"
	"github.com/datvtph41107/bdspro/services/cron"
)

const (
	retentionBatchSize  int32 = 1000
	retentionMaxBatches       = 20
)

// DeleteBatch matches the model's one-batch SQLC operation.
type DeleteBatch func(context.Context, time.Time, int32) (int64, error)

// Sweep deletes at most 20,000 events per run. It is bounded even when
// backlog is large, and is safe to schedule on multiple processes because
// SQL uses FOR UPDATE SKIP LOCKED.
func Sweep(ctx context.Context, days int, deleteBatch DeleteBatch, now time.Time) (int64, error) {
	if ctx == nil {
		return 0, errors.New("audit retention: nil context")
	}
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if days < 1 || days > 3650 || deleteBatch == nil || now.IsZero() {
		return 0, errors.New("audit retention: invalid configuration")
	}
	cutoff := now.UTC().AddDate(0, 0, -days)
	var total int64
	for i := 0; i < retentionMaxBatches; i++ {
		if err := ctx.Err(); err != nil {
			return total, err
		}
		removed, err := deleteBatch(ctx, cutoff, retentionBatchSize)
		if err != nil {
			return total, fmt.Errorf("audit retention: delete: %w", err)
		}
		if removed < 0 || removed > int64(retentionBatchSize) {
			return total, fmt.Errorf("audit retention: invalid rows affected %d", removed)
		}
		total += removed
		if removed < int64(retentionBatchSize) {
			break
		}
	}
	return total, nil
}

// RegisterRetention adds an hourly job to the existing graceful-aware cron
// scheduler. Registration does not start a new goroutine or worker.
func RegisterRetention(s *cron.Scheduler, days int) error {
	if s == nil || days < 1 || days > 3650 {
		return errors.New("audit retention: invalid scheduler or retention days")
	}
	return s.Register("audit-retention", "@hourly", func(ctx context.Context) {
		_, err := Sweep(ctx, days, auditmodel.DeleteOld, time.Now())
		if err != nil && ctx.Err() == nil {
			log.Error("audit retention failed: %v", err)
		}
	})
}
