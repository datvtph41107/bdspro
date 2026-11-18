// Package audit owns persistence of security and business audit evidence.
package audit

import (
	"context"
	"fmt"
	"time"

	"github.com/datvtph41107/bdspro/models/db"
)

// DeleteOld deletes one bounded batch, using the canonical SQLC query.
// The caller controls policy and schedule; this model does not run a worker.
func DeleteOld(ctx context.Context, cutoff time.Time, batchSize int32) (int64, error) {
	if ctx == nil {
		return 0, fmt.Errorf("audit: nil context")
	}
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if cutoff.IsZero() || batchSize < 1 || batchSize > 1000 {
		return 0, fmt.Errorf("audit: invalid prune request")
	}
	q, err := db.GetQueries(ctx)
	if err != nil {
		return 0, err
	}
	return q.DeleteOldAuditEvents(ctx, db.DeleteOldAuditEventsParams{
		Cutoff: cutoff,
		Take:   batchSize,
	})
}
