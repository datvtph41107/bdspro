package queue

import "errors"

// ErrQueueClosed indicates that the process-owned queue manager no longer
// accepts worker registrations after shutdown.
var ErrQueueClosed = errors.New("queue is closed")

// ManagedQueue is a process-owned worker with explicit shutdown.
type ManagedQueue interface {
	Shutdown()
}
