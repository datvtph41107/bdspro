package queue

import (
	"context"
	"errors"

	"github.com/hibiken/asynq"
)

var (
	ErrQueueClosed        = errors.New("queue is closed")
	ErrQueueFull          = errors.New("queue is full")
	ErrUnsupportedOptions = errors.New("queue backend does not support Asynq options")
	ErrUnknownTask        = errors.New("queue task handler not registered")
)

// baseQueue is the submission boundary shared by Redis and channel adapters.
// It does not promise equivalent durability or retry behavior.
type baseQueue interface {
	Enqueue(context.Context, *asynq.Task, ...asynq.Option) error
	Close() error
}

// ManagedQueue is a process-owned worker with explicit shutdown.
type ManagedQueue interface {
	Shutdown()
}
