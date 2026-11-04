package queue

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/hibiken/asynq"
)

func TestChannelQueueCapacityAndClose(t *testing.T) {
	q, err := newBaseChannel(1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := q.Enqueue(ctx, asynq.NewTask("job", []byte("one"))); err != nil {
		t.Fatal(err)
	}
	if err := q.Enqueue(ctx, asynq.NewTask("job", []byte("two"))); !errors.Is(err, ErrQueueFull) {
		t.Fatalf("expected bounded queue: %v", err)
	}
	if q.Len() != 1 {
		t.Fatal("queued item missing")
	}
	task, err := q.Pop(ctx)
	if err != nil || string(task.Payload()) != "one" {
		t.Fatalf("FIFO violated: %v", err)
	}
	if err := q.Close(); err != nil {
		t.Fatal(err)
	}
	if err := q.Enqueue(ctx, asynq.NewTask("job", nil)); !errors.Is(err, ErrQueueClosed) {
		t.Fatalf("closed queue accepted task: %v", err)
	}
}

func TestChannelQueueExecutesHandler(t *testing.T) {
	q, err := NewChannelTaskQueue(4, 2)
	if err != nil {
		t.Fatal(err)
	}
	calls := make(chan string, 2)
	if err := q.Handle("report", asynq.HandlerFunc(func(_ context.Context, task *asynq.Task) error {
		calls <- string(task.Payload())
		return nil
	})); err != nil {
		t.Fatal(err)
	}
	if err := q.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer q.Shutdown()
	if err := q.Enqueue(context.Background(), asynq.NewTask("report", []byte("first"))); err != nil {
		t.Fatal(err)
	}
	select {
	case got := <-calls:
		if got != "first" {
			t.Fatalf("handler got %q", got)
		}
	case <-time.After(time.Second):
		t.Fatal("channel queue did not execute work")
	}
	if err := q.Enqueue(context.Background(), asynq.NewTask("unknown", nil)); !errors.Is(err, ErrUnknownTask) {
		t.Fatalf("unknown task accepted: %v", err)
	}
}

func TestChannelConcurrentPushAndShutdown(t *testing.T) {
	q, err := NewChannelTaskQueue(32, 3)
	if err != nil {
		t.Fatal(err)
	}
	if err := q.Handle("work", asynq.HandlerFunc(func(context.Context, *asynq.Task) error {
		return nil
	})); err != nil {
		t.Fatal(err)
	}
	if err := q.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				_ = q.Enqueue(context.Background(), asynq.NewTask("work", nil))
			}
		}()
	}
	q.Shutdown()
	wg.Wait()
}
