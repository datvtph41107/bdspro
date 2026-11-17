package cmd

import (
	"errors"

	worker_service "github.com/datvtph41107/bdspro/services/worker"
)

func newWorkerHandlerDependencies() (
	worker_service.HandlerDependencies,
	error,
) {
	return worker_service.HandlerDependencies{},
		errors.New(
			"worker: business delivery dependencies not configured",
		)
}
