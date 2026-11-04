package queue

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hibiken/asynq"
)

func TestChannelShutdownContextIsBounded(t *testing.T) {
	q, err := NewChannelTaskQueue(2, 1)
	if err != nil {
		t.Fatal(err)
	}
	started := make(chan struct{})
	release := make(chan struct{})
	defer func() {
		select {
		case <-release:
		default:
			close(release)
		}
		q.Shutdown()
	}()
	if err := q.Handle("blocked", asynq.HandlerFunc(func(context.Context, *asynq.Task) error {
		close(started)
		<-release
		return nil
	})); err != nil {
		t.Fatal(err)
	}
	if err := q.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := q.Enqueue(context.Background(), asynq.NewTask("blocked", nil)); err != nil {
		t.Fatal(err)
	}
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("worker did not start task")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if err := q.ShutdownContext(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("shutdown did not honor deadline: %v", err)
	}
	close(release)
	if err := q.ShutdownContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := q.Enqueue(context.Background(), asynq.NewTask("blocked", nil)); !errors.Is(err, ErrQueueClosed) {
		t.Fatalf("closed queue accepted work: %v", err)
	}
	if err := q.Start(context.Background()); !errors.Is(err, ErrQueueClosed) {
		t.Fatalf("closed queue restarted: %v", err)
	}
}

func TestChannelParentCancellationClosesAdmission(t *testing.T) {
	q, err := NewChannelTaskQueue(2, 1)
	if err != nil {
		t.Fatal(err)
	}
	if err := q.Handle("task", asynq.HandlerFunc(func(context.Context, *asynq.Task) error {
		return nil
	})); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	if err := q.Start(ctx); err != nil {
		t.Fatal(err)
	}
	cancel()
	if err := q.Enqueue(context.Background(), asynq.NewTask("task", nil)); !errors.Is(err, ErrQueueClosed) {
		t.Fatalf("work accepted after parent shutdown: %v", err)
	}
	q.Shutdown()
}
