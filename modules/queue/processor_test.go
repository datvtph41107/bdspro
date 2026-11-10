package queue

import (
	"context"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
)

func TestTaskRegistrationSupportsIndependentHandlers(t *testing.T) {
	p := NewRedisTaskProcessor(asynq.RedisClientOpt{Addr: "127.0.0.1:6379"})
	var emailCalls, reportCalls int
	p.Handle("email.verify.test", asynq.HandlerFunc(func(context.Context, *asynq.Task) error {
		emailCalls++
		return nil
	}))
	p.Handle("report.generate.test", asynq.HandlerFunc(func(context.Context, *asynq.Task) error {
		reportCalls++
		return nil
	}))
	for _, name := range []string{"email.verify.test", "report.generate.test"} {
		if err := p.mux.ProcessTask(context.Background(), asynq.NewTask(name, nil)); err != nil {
			t.Fatal(err)
		}
	}
	if emailCalls != 1 || reportCalls != 1 {
		t.Fatalf("email=%d report=%d", emailCalls, reportCalls)
	}
}


// Real Asynq worker drain: cmd/worker.go closes the database and producer
// after queue.Manager.Shutdown. Active handlers must finish before Shutdown
// returns; this protects their borrowed DB dependencies. This test does not
// claim to forcibly terminate a handler which ignores cancellation forever.
func TestRedisProcessorShutdownWaitsForActiveTask(t *testing.T) {
	redisAddress := os.Getenv("BDS_TEST_REDIS_ADDR")
	if redisAddress == "" {
		t.Skip("requires disposable Redis")
	}
	if os.Getenv("BDS_TEST_MODE") != "disposable" {
		t.Fatal("refusing worker lifecycle test outside disposable mode")
	}

	options := asynq.RedisClientOpt{Addr: redisAddress, DB: 15}
	queueName := "drain_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	processor, err := NewRedisTaskProcessorWithConfig(options, Config{
		Concurrency: 1, Queues: map[string]int{queueName: 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	producer := NewRedisEnqueuer(options)
	inspector := asynq.NewInspector(options)

	started := make(chan struct{})
	release := make(chan struct{})
	completed := make(chan struct{})
	var once sync.Once
	releaseHandler := func() { once.Do(func() { close(release) }) }
	t.Cleanup(func() {
		releaseHandler()
		processor.Shutdown()
		_ = producer.Close()
		_ = inspector.DeleteQueue(queueName, true)
		_ = inspector.Close()
	})

	processor.Handle("test:shutdown-drain", asynq.HandlerFunc(func(_ context.Context, _ *asynq.Task) error {
		close(started)
		<-release
		close(completed)
		return nil
	}))
	if err := processor.Start(); err != nil {
		t.Fatalf("start Asynq processor: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := producer.Enqueue(ctx, asynq.NewTask("test:shutdown-drain", nil),
		asynq.Queue(queueName), asynq.TaskID("drain-"+uuid.NewString())); err != nil {
		t.Fatalf("enqueue task: %v", err)
	}
	select {
	case <-started:
	case <-ctx.Done():
		t.Fatalf("Asynq worker did not begin task: %v", ctx.Err())
	}

	shutdownDone := make(chan struct{})
	go func() {
		processor.Shutdown()
		close(shutdownDone)
	}()
	select {
	case <-shutdownDone:
		t.Fatal("worker shutdown completed while a task still owned its dependencies")
	case <-time.After(150 * time.Millisecond):
	}

	releaseHandler()
	select {
	case <-shutdownDone:
	case <-ctx.Done():
		t.Fatalf("Asynq worker did not drain after the task completed: %v", ctx.Err())
	}
	select {
	case <-completed:
	default:
		t.Fatal("worker shutdown returned before task handler completion")
	}
}
