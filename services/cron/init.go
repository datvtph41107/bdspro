package cron

import (
	"context"
	"fmt"

	"github.com/datvtph41107/bdspro/modules/graceful"
)

// RegisterTasks explicitly registers jobs owned by
// business services. No job starts during registration.
type RegisterTasks func(*Scheduler) error

// Init prepares the cron scheduler and runs it under
// the process-wide graceful lifecycle.
//
// It blocks while the scheduler is running.
func Init(
	manager *graceful.Manager,
	register RegisterTasks,
) error {
	if manager == nil || register == nil {
		return fmt.Errorf(
			"cron: manager and task registration are required",
		)
	}

	scheduler := New(manager.ShutdownContext())

	if err := register(scheduler); err != nil {
		return fmt.Errorf(
			"register cron tasks: %w", err,
		)
	}

	if !scheduler.HasJobs() {
		return nil
	}

	return manager.RunWithCancel(scheduler)
}

// StopRequested reports whether this process has
// begun its graceful shutdown phase.
func StopRequested(ctx context.Context) bool {
	return ctx.Err() != nil
}
