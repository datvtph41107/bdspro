package graceful

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/datvtph41107/bdspro/modules/startup"
)

func TestManagerWaitsForActivityBeforeResourceTermination(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	g := newGracefulManager(ctx, 3*time.Second)
	var cleaned atomic.Int32
	g.RunAtTerminate(func() { cleaned.Add(1) })
	if err := g.RegisterServer(); err != nil {
		t.Fatal(err)
	}
	g.DoGracefulShutdown()
	select {
	case <-g.Done():
		t.Fatal("process terminated while startup/server activity was registered")
	default:
	}
	g.ServerDone()
	select {
	case <-g.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("graceful shutdown did not finish after activity ended")
	}
	if got := cleaned.Load(); got != 1 {
		t.Fatalf("termination callback called %d times", got)
	}
	g.DoGracefulShutdown()
	if got := cleaned.Load(); got != 1 {
		t.Fatal("termination callback was not idempotent")
	}
}

// A shutdown signal must cancel ongoing bootstrap work before the hammer
// deadline. The Manager must still wait for that activity to finish.
func TestShutdownContextCancelsStartupBeforeHammer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	g := newGracefulManager(ctx, 3*time.Second)
	if err := g.RegisterServer(); err != nil {
		t.Fatal(err)
	}
	entered := make(chan struct{})
	completed := make(chan error, 1)
	go func() {
		defer g.ServerDone()
		completed <- startup.Run(g.ShutdownContext(), startup.Step{
			Name: "PostgreSQL",
			Init: func(ctx context.Context) error {
				close(entered)
				<-ctx.Done()
				return ctx.Err()
			},
		})
	}()
	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("bootstrap step did not begin")
	}
	g.DoGracefulShutdown()
	select {
	case err := <-completed:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected wrapped context cancellation, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("startup waited for hammer instead of honoring shutdown")
	}
	select {
	case <-g.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("Manager did not complete after startup exited")
	}
}
