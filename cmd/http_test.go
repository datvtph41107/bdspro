package cmd

import (
	"context"
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
