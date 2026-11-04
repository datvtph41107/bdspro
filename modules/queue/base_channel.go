package queue

import (
	"context"
	"fmt"
	"sync"

	"github.com/hibiken/asynq"
)

// baseChannel is bounded, in-memory and intentionally non-durable.
// It is useful for local processing and tests, not committed SQL outbox jobs.
type baseChannel struct {
	mu     sync.Mutex
	tasks  chan *asynq.Task
	closed bool
}

var _ baseQueue = (*baseChannel)(nil)

func newBaseChannel(capacity int) (*baseChannel, error) {
	if capacity < 1 {
		return nil, fmt.Errorf("channel queue capacity must be positive")
	}
	return &baseChannel{tasks: make(chan *asynq.Task, capacity)}, nil
}

func (q *baseChannel) Enqueue(ctx context.Context, task *asynq.Task, opts ...asynq.Option) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if task == nil {
		return fmt.Errorf("channel queue task is nil")
	}
	if len(opts) != 0 {
		return ErrUnsupportedOptions
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed {
		return ErrQueueClosed
	}
	select {
	case q.tasks <- task:
		return nil
	default:
		return ErrQueueFull
	}
}

func (q *baseChannel) Pop(ctx context.Context) (*asynq.Task, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case task := <-q.tasks:
		return task, nil
	}
}

func (q *baseChannel) Close() error {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.closed = true
	return nil
}

func (q *baseChannel) Len() int {
	return len(q.tasks)
}
