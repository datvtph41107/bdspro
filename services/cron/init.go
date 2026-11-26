package cron

import (
	"context"
	"fmt"

	"github.com/datvtph41107/bdspro/modules/graceful"
)

type RegisterTasks func(*Scheduler) error

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

func StopRequested(ctx context.Context) bool {
	return ctx.Err() != nil
}
