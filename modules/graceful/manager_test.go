package graceful

import (
	"context"
	"testing"
	"time"
)

func TestShutdownDrainAndTerminate(t *testing.T) {
	g := newManager(context.Background(), time.Second)
	g.RegisterServer()
	ended := make(chan struct{})
	g.RunAtTerminate(func() { close(ended) })
	g.DoGracefulShutdown()
	select {
	case <-g.IsShutdown():
	default:
		t.Fatal("shutdown context was not canceled")
	}
	select {
	case <-g.Done():
		t.Fatal("finished while server is running")
	default:
	}
	g.ServerDone()
	select {
	case <-g.Done():
	case <-time.After(time.Second):
		t.Fatal("graceful manager did not finish")
	}
	select {
	case <-ended:
	default:
		t.Fatal("terminate hook was not waited for")
	}
}

func TestShutdownTimeoutHammersRunningServer(t *testing.T) {
	g := newManager(context.Background(), 10*time.Millisecond)
	g.RegisterServer()
	g.DoGracefulShutdown()
	select {
	case <-g.IsHammer():
	case <-time.After(time.Second):
		t.Fatal("hammer deadline was not enforced")
	}
	g.ServerDone()
	<-g.Done()
}

func TestParentCancelDoesNotSkipDrain(t *testing.T) {
	parent, cancel := context.WithCancel(context.Background())
	g := newManager(parent, time.Second)
	g.RegisterServer()
	cancel()
	select {
	case <-g.IsShutdown():
	case <-time.After(time.Second):
		t.Fatal("parent cancellation did not initiate shutdown")
	}
	select {
	case <-g.IsHammer():
		t.Fatal("hammer happened before drain deadline")
	default:
	}
	select {
	case <-g.TerminateContext().Done():
		t.Fatal("termination happened before running server completed")
	default:
	}
	g.ServerDone()
	select {
	case <-g.Done():
	case <-time.After(time.Second):
		t.Fatal("shutdown did not finish")
	}
}
