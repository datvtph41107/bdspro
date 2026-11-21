package graceful

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"
)

func TestHTTPShutdownDrainsActiveRequestBeforeResourceTermination(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	manager := newGracefulManager(ctx, 3*time.Second)
	active := make(chan struct{})
	release := make(chan struct{})
	var releaseOnce sync.Once
	defer releaseOnce.Do(func() { close(release) })
	resourcesClosed := make(chan struct{})
	manager.RunAtTerminate(func() { close(resourcesClosed) })

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(active)
		<-release
		w.WriteHeader(http.StatusNoContent)
	})
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server, serve, err := newHTTPServer(
		manager, "tcp", listener.Addr().String(), "drain-test", handler, time.Second,
	)
	if err != nil {
		_ = listener.Close()
		t.Fatal(err)
	}
	server.listener = newWrappedListener(listener, server)
	serveDone := make(chan error, 1)
	go func() { serveDone <- server.Serve(serve) }()
	requestDone := make(chan error, 1)
	go func() {
		client := &http.Client{Timeout: 3 * time.Second}
		resp, err := client.Get("http://" + listener.Addr().String())
		if err == nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
			if resp.StatusCode != http.StatusNoContent {
				err = errors.New("unexpected HTTP response status")
			}
		}
		requestDone <- err
	}()
	select {
	case <-active:
	case <-time.After(2 * time.Second):
		t.Fatal("HTTP handler never started")
	}
	manager.DoGracefulShutdown()
	select {
	case <-resourcesClosed:
		t.Fatal("resources terminated while request was active")
	case <-time.After(50 * time.Millisecond):
	}
	releaseOnce.Do(func() { close(release) })
	select {
	case err := <-requestDone:
		if err != nil {
			t.Fatalf("in-flight HTTP request interrupted: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("request did not drain")
	}
	select {
	case err := <-serveDone:
		if err != nil {
			t.Fatalf("HTTP serve did not complete cleanly: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("HTTP server did not stop")
	}
	select {
	case <-resourcesClosed:
	case <-time.After(2 * time.Second):
		t.Fatal("termination cleanup never ran")
	}
}

func TestHTTPShutdownTimeoutClosesBlockedConnection(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	manager := newGracefulManager(ctx, 3*time.Second)
	active := make(chan struct{})
	release := make(chan struct{})
	exited := make(chan struct{})
	var once sync.Once
	defer once.Do(func() { close(release) })
	handler := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		defer close(exited)
		close(active)
		<-release // a noncooperative handler may outlive the shutdown deadline
	})
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server, serve, err := newHTTPServer(
		manager, "tcp", listener.Addr().String(), "deadline-test", handler, 80*time.Millisecond,
	)
	if err != nil {
		_ = listener.Close()
		t.Fatal(err)
	}
	server.listener = newWrappedListener(listener, server)
	serveDone := make(chan error, 1)
	go func() { serveDone <- server.Serve(serve) }()
	clientDone := make(chan error, 1)
	go func() {
		client := &http.Client{Timeout: 3 * time.Second}
		resp, err := client.Get("http://" + listener.Addr().String())
		if resp != nil {
			_ = resp.Body.Close()
		}
		clientDone <- err
	}()
	select {
	case <-active:
	case <-time.After(2 * time.Second):
		t.Fatal("HTTP handler never started")
	}
	manager.DoGracefulShutdown()
	select {
	case err := <-clientDone:
		if err == nil {
			t.Fatal("blocked HTTP connection should close at drain deadline")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("HTTP shutdown did not honor its deadline")
	}
	select {
	case <-serveDone:
	case <-time.After(2 * time.Second):
		t.Fatal("HTTP serve stuck after forced close")
	}
	select {
	case <-manager.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("termination did not finish after forced close")
	}
	once.Do(func() { close(release) })
	select {
	case <-exited:
	case <-time.After(2 * time.Second):
		t.Fatal("test handler did not exit")
	}
}
