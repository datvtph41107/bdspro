package queue

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/datvtph41107/bdspro/modules/log"
	"github.com/hibiken/asynq"
)

// ChannelTaskQueue combines a bounded channel with local worker goroutines.
// It deliberately does not implement persistence, scheduled delivery, retry,
// task leasing or distributed execution. Use Asynq for durable workloads.
type ChannelTaskQueue struct {
	base     *baseChannel
	mu       sync.Mutex
	handlers map[string]asynq.Handler
	workers  int
	started  bool
	closed   bool
	runCtx   context.Context
	cancel   context.CancelFunc
	done     chan struct{}
}

func NewChannelTaskQueue(capacity, concurrency int) (*ChannelTaskQueue, error) {
	if concurrency < 1 {
		return nil, fmt.Errorf("channel queue concurrency must be positive")
	}
	base, err := newBaseChannel(capacity)
	if err != nil {
		return nil, err
	}
	return &ChannelTaskQueue{
		base: base, workers: concurrency,
		handlers: make(map[string]asynq.Handler),
	}, nil
}

func (q *ChannelTaskQueue) Handle(taskType string, handler asynq.Handler) error {
	if q == nil || handler == nil || taskType == "" {
		return errors.New("queue handler and type are required")
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed {
		return ErrQueueClosed
	}
	if q.started {
		return errors.New("cannot register handler after queue startup")
	}
	if _, exists := q.handlers[taskType]; exists {
		return fmt.Errorf("task handler already registered: %q", taskType)
	}
	q.handlers[taskType] = handler
	return nil
}

func (q *ChannelTaskQueue) Enqueue(ctx context.Context, task *asynq.Task, opts ...asynq.Option) error {
	if q == nil {
		return ErrQueueClosed
	}
	if task == nil {
		return errors.New("channel queue task is nil")
	}
	q.mu.Lock()
	handler := q.handlers[task.Type()]
	runCtx, closed := q.runCtx, q.closed
	q.mu.Unlock()
	if closed || (runCtx != nil && runCtx.Err() != nil) {
		return ErrQueueClosed
	}
	if handler == nil {
		return ErrUnknownTask
	}
	return q.base.Enqueue(ctx, task, opts...)
}

func (q *ChannelTaskQueue) Start(ctx context.Context) error {
	if q == nil {
		return ErrQueueClosed
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed {
		return ErrQueueClosed
	}
	if q.started {
		return errors.New("channel queue already started")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	q.started = true
	runCtx, cancel := context.WithCancel(ctx)
	q.cancel = cancel
	q.runCtx = runCtx
	q.done = make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < q.workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				task, err := q.base.Pop(runCtx)
				if err != nil {
					return
				}
				if err := q.handlers[task.Type()].ProcessTask(runCtx, task); err != nil {
					log.Ctx(runCtx).Error().Str("task", task.Type()).
						Str("error_type", fmt.Sprintf("%T", err)).
						Msg("queue.channel_task_failed")
				}
			}
		}()
	}
	go func() { wg.Wait(); close(q.done) }()
	return nil
}

// ShutdownContext stops accepting work, cancels workers and awaits them.
// A handler ignoring its context cannot be forcibly terminated.
func (q *ChannelTaskQueue) ShutdownContext(ctx context.Context) error {
	if q == nil {
		return nil
	}
	q.mu.Lock()
	q.closed = true
	cancel, done := q.cancel, q.done
	_ = q.base.Close()
	q.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if done == nil {
		return nil
	}
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Shutdown provides a bounded timeout to the process queue Manager.
func (q *ChannelTaskQueue) Shutdown() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := q.ShutdownContext(ctx); err != nil {
		log.Error("channel worker shutdown timed out: %v", err)
	}
}

func (q *ChannelTaskQueue) Len() int {
	return q.base.Len()
}
