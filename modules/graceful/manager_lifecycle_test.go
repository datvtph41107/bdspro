package graceful

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
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
