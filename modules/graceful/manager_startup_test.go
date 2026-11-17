package graceful

import (
	"context"
	"errors"
	"sync"
	"testing"
)

func TestG019StopBeforeRegistration(t *testing.T) {
	g := newGracefulManager(context.Background(), -1)
	g.DoGracefulShutdown()
	await(t, g.IsShutdown(), "shutdown")
	if err := g.RegisterServer(); !errors.Is(err, ErrShuttingDown) {
		t.Fatalf("late registration = %v; want ErrShuttingDown", err)
	}
	await(t, g.Done(), "done")
}

func TestG019StopWhileStartupRegistered(t *testing.T) {
	g := newGracefulManager(context.Background(), -1)
	if err := g.RegisterServer(); err != nil {
		t.Fatal(err)
	}
	terminated := make(chan struct{})
	g.RunAtTerminate(func() { close(terminated) })
	g.DoGracefulShutdown()
	select {
	case <-g.Done():
		t.Fatal("manager finished during startup")
	default:
	}
	g.ServerDone()
	await(t, terminated, "terminate")
	await(t, g.Done(), "done")
}

func TestG019ConcurrentRegistrationAndShutdown(t *testing.T) {
	g := newGracefulManager(context.Background(), -1)
	const n = 100
	release := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := g.RegisterServer()
			if err == nil {
				<-release
				g.ServerDone()
				return
			}
			if !errors.Is(err, ErrShuttingDown) {
				t.Errorf("unexpected: %v", err)
			}
		}()
	}
	g.DoGracefulShutdown()
	g.DoGracefulShutdown()
	close(release)
	wg.Wait()
	await(t, g.Done(), "done")
}

func TestG019ActivityRejectedAfterShutdown(t *testing.T) {
	g := newGracefulManager(context.Background(), -1)
	g.DoGracefulShutdown()
	ran := false
	err := g.RunWithShutdownContext(func(context.Context) { ran = true })
	if !errors.Is(err, ErrShuttingDown) || ran {
		t.Fatalf("ran=%v err=%v", ran, err)
	}
	await(t, g.Done(), "done")
}
