package worker

import (
	"context"
	"fmt"

	"github.com/datvtph41107/bdspro/services/cron"
)

// RegisterOutboxSchedule connects the dispatcher to
// the scheduler. Scheduling stays separate from dispatch.
func RegisterOutboxSchedule(
	scheduler *cron.Scheduler,
	dispatcher *OutboxDispatcher,
) error {
	if scheduler == nil || dispatcher == nil {
		return fmt.Errorf(
			"worker: scheduler and dispatcher are required",
		)
	}

	return scheduler.Register(
		"dispatch-outbox",
		"* * * * *",
		func(ctx context.Context) {
			if ctx.Err() != nil {
				return
			}

			// Production implementation must log errors
			// without logging task payloads.
			_, _ = dispatcher.Dispatch(ctx)
		},
	)
}
