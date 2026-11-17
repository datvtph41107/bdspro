package graceful

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func httpTestAwait[T any](t *testing.T, ch <-chan T, label string) T {
	t.Helper()
	select {
	case x := <-ch:
		return x
	case <-time.After(5 * time.Second):
		t.Fatalf("timeout waiting for %s", label)
		var zero T
		return zero
	}
}

func TestHTTPStartsAndStopsWithoutConnections(t *testing.T) {
	g := newGracefulManager(context.Background(), time.Second)
	listening := make(chan net.Addr, 1)
	result := make(chan error, 1)
	go func() {
		result <- HTTPListenAndServe(g, "tcp", "127.0.0.1:0", "test", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}), time.Second, func(addr net.Addr) { listening <- addr })
	}()
	addr := httpTestAwait(t, listening, "HTTP readiness")
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("http://" + addr.String() + "/health")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status=%d want 204", resp.StatusCode)
	}
	g.DoGracefulShutdown()
	if err := httpTestAwait(t, result, "server stop"); err != nil {
		t.Fatal(err)
	}
	httpTestAwait(t, g.Done(), "manager done")
}

func TestHTTPDrainBlocksTerminationUntilHandlerReturns(t *testing.T) {
	g := newGracefulManager(context.Background(), 2*time.Second)
	entered := make(chan struct{})
	release := make(chan struct{})
	defer func() {
		select {
		case <-release:
		default:
			close(release)
		}
	}()
	terminated := make(chan struct{})
	g.RunAtTerminate(func() { close(terminated) })
	listening := make(chan net.Addr, 1)
	stopped := make(chan error, 1)
	go func() {
		stopped <- HTTPListenAndServe(g, "tcp", "127.0.0.1:0", "test", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			close(entered)
			<-release
			w.WriteHeader(http.StatusNoContent)
		}), time.Second, func(addr net.Addr) { listening <- addr })
	}()
	addr := httpTestAwait(t, listening, "listener")
	requestDone := make(chan error, 1)
	go func() {
		client := &http.Client{Timeout: 3 * time.Second}
		resp, err := client.Get("http://" + addr.String())
		if err == nil {
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusNoContent {
				err = fmt.Errorf("status=%d", resp.StatusCode)
			}
		}
		requestDone <- err
	}()
	httpTestAwait(t, entered, "handler begin")
	g.DoGracefulShutdown()
	httpTestAwait(t, g.IsShutdown(), "shutdown phase")
	select {
	case err := <-stopped:
		t.Fatalf("HTTP completed early: %v", err)
	case <-terminated:
		t.Fatal("DB cleanup started while handler still running")
	case <-time.After(120 * time.Millisecond):
	}
	close(release)
	if err := httpTestAwait(t, requestDone, "HTTP response"); err != nil {
		t.Fatal(err)
	}
	if err := httpTestAwait(t, stopped, "server drain"); err != nil {
		t.Fatal(err)
	}
	httpTestAwait(t, terminated, "terminate hook")
	httpTestAwait(t, g.Done(), "manager complete")
}

func TestHTTPShutdownDeadlineForcesConnectionButStillWaitsHandler(t *testing.T) {
	g := newGracefulManager(context.Background(), time.Second)
	entered := make(chan struct{})
	release := make(chan struct{})
	defer func() {
		select {
		case <-release:
		default:
			close(release)
		}
	}()
	listening := make(chan net.Addr, 1)
	stopped := make(chan error, 1)
	go func() {
		stopped <- HTTPListenAndServe(g, "tcp", "127.0.0.1:0", "slow", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			close(entered)
			<-release // Ignoring cancellation is a deliberately uncooperative handler.
			w.WriteHeader(http.StatusNoContent)
		}), 70*time.Millisecond, func(addr net.Addr) { listening <- addr })
	}()
	addr := httpTestAwait(t, listening, "listener")
	clientDone := make(chan error, 1)
	go func() {
		client := &http.Client{Timeout: 4 * time.Second}
		resp, err := client.Get("http://" + addr.String())
		if err == nil {
			resp.Body.Close()
			err = fmt.Errorf("unexpected response: %d", resp.StatusCode)
		}
		clientDone <- err
	}()
	httpTestAwait(t, entered, "handler begin")
	g.DoGracefulShutdown()
	if err := httpTestAwait(t, clientDone, "connection close on deadline"); err == nil {
		t.Fatal("expected forced client connection error")
	}
	select {
	case err := <-stopped:
		t.Fatalf("ServerDone before handler return: %v", err)
	case <-g.Done():
		t.Fatal("Manager.Done before handler return")
	case <-time.After(80 * time.Millisecond):
	}
	close(release)
	if err := httpTestAwait(t, stopped, "drain after forced close"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("want shutdown deadline error, got %v", err)
	}
	httpTestAwait(t, g.Done(), "manager done")
}

func TestHTTPStopBeforeListenerAndBadAddress(t *testing.T) {
	g := newGracefulManager(context.Background(), -1)
	g.DoGracefulShutdown()
	called := false
	err := HTTPListenAndServe(g, "tcp", "127.0.0.1:0", "test", http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}), time.Second, func(net.Addr) { called = true })
	if !errors.Is(err, ErrShuttingDown) || called {
		t.Fatalf("shutdown startup: err=%v ready=%v", err, called)
	}
	httpTestAwait(t, g.Done(), "done")

	g2 := newGracefulManager(context.Background(), -1)
	err = HTTPListenAndServe(g2, "tcp", "127.0.0.1:-1", "test", http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}), time.Second, nil)
	if err == nil || !strings.Contains(err.Error(), "listen HTTP") {
		t.Fatalf("want listener error, got %v", err)
	}
	g2.DoGracefulShutdown()
	httpTestAwait(t, g2.Done(), "done")
}

func TestActiveHTTPHandlersDrainUntilHandlerReturns(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})
	active := &activeHTTPHandlers{next: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		close(entered)
		<-release
		w.WriteHeader(http.StatusNoContent)
	})}
	done := make(chan struct{})
	go func() {
		defer close(done)
		active.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	}()
	httpTestAwait(t, entered, "handler")
	drained := make(chan struct{})
	go func() { active.Drain(); close(drained) }()
	select {
	case <-drained:
		t.Fatal("drained before handler completed")
	case <-time.After(50 * time.Millisecond):
	}
	close(release)
	httpTestAwait(t, drained, "drain")
	httpTestAwait(t, done, "handler return")
	rec := httptest.NewRecorder()
	active.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("late request status=%d", rec.Code)
	}
}
