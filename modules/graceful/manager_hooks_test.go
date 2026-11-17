package graceful

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestShutdownHookWaitsBeforeTerminate(t *testing.T) {
	g := newGracefulManager(context.Background(), -1)
	started, release := make(chan struct{}), make(chan struct{})
	g.RunAtShutdown(context.Background(), func() { close(started); <-release })
	terminated := make(chan struct{})
	g.RunAtTerminate(func() { close(terminated) })
	g.DoGracefulShutdown()
	httpTestAwait(t, started, "shutdown hook start")
	select {
	case <-terminated:
		t.Fatal("cleanup before shutdown callback returned")
	default:
	}
	close(release)
	httpTestAwait(t, terminated, "cleanup")
	httpTestAwait(t, g.Done(), "done")
}

func TestPanicInHooksDoesNotPreventOtherCleanup(t *testing.T) {
	g := newGracefulManager(context.Background(), -1)
	g.RunAtShutdown(context.Background(), func() { panic("shutdown panic") })
	g.RunAtTerminate(func() { panic("termination panic") })
	completed := make(chan struct{})
	g.RunAtTerminate(func() { close(completed) })
	g.DoGracefulShutdown()
	httpTestAwait(t, completed, "other cleanup")
	httpTestAwait(t, g.Done(), "done")
}

type testManagedCanceler struct {
	ready    chan struct{}
	stop     chan struct{}
	stopOnce sync.Once
}

func (w *testManagedCanceler) Run()    { close(w.ready); <-w.stop }
func (w *testManagedCanceler) Cancel() { w.stopOnce.Do(func() { close(w.stop) }) }

func TestRunWithCancelAndLateRegistration(t *testing.T) {
	g := newGracefulManager(context.Background(), -1)
	w := &testManagedCanceler{ready: make(chan struct{}), stop: make(chan struct{})}
	runDone := make(chan error, 1)
	go func() { runDone <- g.RunWithCancel(w) }()
	httpTestAwait(t, w.ready, "worker startup")
	g.DoGracefulShutdown()
	if err := httpTestAwait(t, runDone, "worker stop"); err != nil {
		t.Fatal(err)
	}
	httpTestAwait(t, g.Done(), "done")
	if err := g.RunWithCancel(w); err != ErrShuttingDown {
		t.Fatalf("late activity=%v", err)
	}
}

func TestHammerDeadlineStopsAfterDrainNotDuringTerminate(t *testing.T) {
	g := newGracefulManager(context.Background(), 70*time.Millisecond)
	blocking := make(chan struct{})
	g.RunAtTerminate(func() { <-blocking })
	g.DoGracefulShutdown()
	httpTestAwait(t, g.TerminateContext().Done(), "terminate phase")
	// Hammer has no reason to fire after all running work drained, even
	// if termination itself takes longer than the hammer timeout.
	time.Sleep(120 * time.Millisecond)
	select {
	case <-g.IsHammer():
		t.Fatal("hammer fired during resource cleanup")
	default:
	}
	close(blocking)
	httpTestAwait(t, g.Done(), "done")
}
