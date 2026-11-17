package graceful

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func await(t *testing.T, ch <-chan struct{}, label string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for %s", label)
	}
}

func TestShutdownDrainAndTerminate(t *testing.T) {
	g := newGracefulManager(context.Background(), time.Second)
	if err := g.RegisterServer(); err != nil {
		t.Fatal(err)
	}
	callbackDone := make(chan struct{})
	allowCallback := make(chan struct{})
	g.RunAtShutdown(context.Background(), func() { <-allowCallback; close(callbackDone) })
	terminateDone := make(chan struct{})
	g.RunAtTerminate(func() { close(terminateDone) })
	g.DoGracefulShutdown()
	await(t, g.IsShutdown(), "shutdown")
	g.ServerDone()
	select {
	case <-g.Done():
		t.Fatal("done before shutdown hook")
	default:
	}
	close(allowCallback)
	await(t, callbackDone, "shutdown hook")
	await(t, g.Done(), "manager done")
	select {
	case <-terminateDone:
	default:
		t.Fatal("terminate hook did not finish")
	}
}

func TestParentCancellationPreservesDrain(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	g := newGracefulManager(ctx, time.Second)
	if err := g.RegisterServer(); err != nil {
		t.Fatal(err)
	}
	cancel()
	await(t, g.IsShutdown(), "shutdown")
	select {
	case <-g.IsHammer():
		t.Fatal("premature hammer")
	default:
	}
	select {
	case <-g.TerminateContext().Done():
		t.Fatal("premature terminate")
	default:
	}
	g.ServerDone()
	await(t, g.Done(), "done")
}

func TestHammerAfterTimeoutDoesNotForceFinish(t *testing.T) {
	g := newGracefulManager(context.Background(), 10*time.Millisecond)
	if err := g.RegisterServer(); err != nil {
		t.Fatal(err)
	}
	g.DoGracefulShutdown()
	await(t, g.IsHammer(), "hammer")
	select {
	case <-g.Done():
		t.Fatal("hammer is not server completion")
	default:
	}
	g.ServerDone()
	await(t, g.Done(), "done")
}

func TestStopBeforeRegistration(t *testing.T) {
	g := newGracefulManager(context.Background(), -1)
	g.DoGracefulShutdown()
	await(t, g.IsShutdown(), "shutdown")
	if err := g.RegisterServer(); !errors.Is(err, ErrShuttingDown) {
		t.Fatalf("late registration = %v, want ErrShuttingDown", err)
	}
	await(t, g.Done(), "done")
}

func TestStopWhileStartupRegistered(t *testing.T) {
	g := newGracefulManager(context.Background(), -1)
	if err := g.RegisterServer(); err != nil {
		t.Fatal(err)
	}
	terminate := make(chan struct{})
	g.RunAtTerminate(func() { close(terminate) })
	g.DoGracefulShutdown()
	select {
	case <-g.Done():
		t.Fatal("done before startup finished")
	default:
	}
	g.ServerDone()
	await(t, terminate, "cleanup")
	await(t, g.Done(), "done")
}

func TestConcurrentRegistrationsAndRepeatedShutdown(t *testing.T) {
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
				t.Errorf("unexpected error: %v", err)
			}
		}()
	}
	g.DoGracefulShutdown()
	g.DoGracefulShutdown()
	close(release)
	wg.Wait()
	await(t, g.Done(), "done")
}

func TestRunWithShutdownContextRejectedAfterShutdown(t *testing.T) {
	g := newGracefulManager(context.Background(), -1)
	g.DoGracefulShutdown()
	ran := false
	err := g.RunWithShutdownContext(func(context.Context) { ran = true })
	if !errors.Is(err, ErrShuttingDown) || ran {
		t.Fatalf("run=%v err=%v", ran, err)
	}
	await(t, g.Done(), "done")
}

func TestTerminateRegistrationDuringDrain(t *testing.T) {
	g := newGracefulManager(context.Background(), -1)
	if err := g.RegisterServer(); err != nil {
		t.Fatal(err)
	}
	g.DoGracefulShutdown()
	terminated := make(chan struct{})
	g.RunAtTerminate(func() { close(terminated) })
	select {
	case <-terminated:
		t.Fatal("early cleanup")
	default:
	}
	g.ServerDone()
	await(t, g.Done(), "done")
	select {
	case <-terminated:
	default:
		t.Fatal("cleanup not run")
	}
}
