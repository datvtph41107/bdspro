package cmd

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestServeHTTPStartupAndShutdown(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	listening := make(chan net.Addr, 1)
	result := make(chan error, 1)
	go func() {
		result <- serveHTTP(ctx, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}), "127.0.0.1:0", func(addr net.Addr) { listening <- addr })
	}()

	var addr net.Addr
	select {
	case addr = <-listening:
	case err := <-result:
		t.Fatalf("HTTP server exited before readiness: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("HTTP server did not report readiness")
	}

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(fmt.Sprintf("http://%s/health", addr.String()))
	if err != nil {
		t.Fatalf("HTTP handler not available after readiness: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("HTTP handler returned %d; want 204", resp.StatusCode)
	}

	cancel()
	select {
	case err := <-result:
		if err != nil {
			t.Fatalf("graceful shutdown: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("HTTP server did not shut down")
	}
}

func TestServeHTTPInvalidListener(t *testing.T) {
	err := serveHTTP(context.Background(), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}), "127.0.0.1:-1", nil)
	if err == nil || !strings.Contains(err.Error(), "listen HTTP") {
		t.Fatalf("invalid listener error = %v", err)
	}
}

func TestServeHTTPPreCanceledContextNeverReportsReadiness(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	called := false
	err := serveHTTP(ctx, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}),
		"127.0.0.1:0", func(net.Addr) { called = true })
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("already canceled startup must be rejected: %v", err)
	}
	if called {
		t.Fatal("canceled startup reported HTTP readiness")
	}
}

func TestShutdownHTTPServerForcesActiveConnectionCloseOnTimeout(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	entered := make(chan struct{})
	release := make(chan struct{})
	defer close(release)
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(entered)
		<-release // Simulate an uncooperative handler that cannot drain promptly.
		w.WriteHeader(http.StatusNoContent)
	})}
	defer server.Close()
	serveDone := make(chan error, 1)
	go func() { serveDone <- server.Serve(listener) }()

	requestDone := make(chan error, 1)
	go func() {
		client := &http.Client{Timeout: 3 * time.Second}
		response, err := client.Get("http://" + listener.Addr().String())
		if err == nil {
			response.Body.Close()
			err = fmt.Errorf("unexpected HTTP response: %d", response.StatusCode)
		}
		requestDone <- err
	}()
	select {
	case <-entered:
	case <-time.After(3 * time.Second):
		t.Fatal("request never reached the blocking handler")
	}
	err = shutdownHTTPServer(server, 80*time.Millisecond)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("shutdown must report graceful drain timeout: %v", err)
	}
	select {
	case err := <-requestDone:
		if err == nil {
			t.Fatal("active connection was not interrupted after shutdown timeout")
		}
	case <-time.After(time.Second):
		t.Fatal("active connection stayed open after shutdown deadline")
	}
	select {
	case err := <-serveDone:
		if !errors.Is(err, http.ErrServerClosed) {
			t.Fatalf("server did not stop accepting connections: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("HTTP Serve goroutine did not exit")
	}
}
