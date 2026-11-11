package graceful

import (
	"context"
	"sync"
	"testing"
	"time"
)

func await(t *testing.T, ch <-chan struct{}, name string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Fatalf("timeout waiting for %s", name)
	}
}
func TestShutdownDrainAndTerminate(t *testing.T) {
	g := newManager(context.Background(), time.Second)
	g.RegisterServer()
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
		t.Fatal("manager finished before shutdown callback")
	default:
	}
	close(allowCallback)
	await(t, callbackDone, "shutdown callback")
	await(t, g.Done(), "done")
	select {
	case <-terminateDone:
	default:
		t.Fatal("termination callback not waited for")
	}
}
func TestHammerDeadlineCooperative(t *testing.T) {
	g := newManager(context.Background(), 10*time.Millisecond)
	g.RegisterServer()
	g.DoGracefulShutdown()
	await(t, g.IsHammer(), "hammer")
	select {
	case <-g.Done():
		t.Fatal("hammer incorrectly bypassed running work")
	default:
	}
	g.ServerDone()
	await(t, g.Done(), "done")
}
func TestParentCancelPreservesDrain(t *testing.T) {
	parent, cancel := context.WithCancel(context.Background())
	g := newManager(parent, time.Second)
	g.RegisterServer()
	cancel()
	await(t, g.IsShutdown(), "shutdown")
	select {
	case <-g.IsHammer():
		t.Fatal("hammer before deadline")
	default:
	}
	select {
	case <-g.TerminateContext().Done():
		t.Fatal("terminate before drain")
	default:
	}
	g.ServerDone()
	await(t, g.Done(), "done")
}
func TestRepeatedShutdownIsIdempotent(t *testing.T) {
	g := newManager(context.Background(), time.Second)
	g.RegisterServer()
	g.DoGracefulShutdown()
	g.DoGracefulShutdown()
	select {
	case <-g.IsHammer():
		t.Fatal("repeated shutdown escalated prematurely")
	default:
	}
	g.ServerDone()
	await(t, g.Done(), "done")
}

type testRunner struct {
	started  chan struct{}
	done     chan struct{}
	canceled chan struct{}
	once     sync.Once
}

func (r *testRunner) Run()    { close(r.started); <-r.canceled; close(r.done) }
func (r *testRunner) Cancel() { r.once.Do(func() { close(r.canceled) }) }
func TestRunWithCancel(t *testing.T) {
	g := newManager(context.Background(), time.Second)
	r := &testRunner{started: make(chan struct{}), done: make(chan struct{}), canceled: make(chan struct{})}
	go g.RunWithCancel(r)
	<-r.started
	g.DoGracefulShutdown()
	await(t, r.done, "runner")
	await(t, g.Done(), "done")
}
