package graceful

import (
	"context"
	"net"
	"net/http"
	"sync"
	"time"
)

// activeHTTPHandlers protects the cleanup boundary even when net/http.Close
// interrupts client connections but a Go handler has not returned yet.
type activeHTTPHandlers struct {
	next    http.Handler
	mu      sync.Mutex
	closing bool
	running sync.WaitGroup
}

func (a *activeHTTPHandlers) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	if a.closing {
		a.mu.Unlock()
		http.Error(w, "server shutting down", http.StatusServiceUnavailable)
		return
	}
	a.running.Add(1)
	a.mu.Unlock()
	defer a.running.Done()
	a.next.ServeHTTP(w, r)
}

func (a *activeHTTPHandlers) Drain() {
	a.mu.Lock()
	a.closing = true
	a.mu.Unlock()
	a.running.Wait()
}

// newHTTPServer adapts Gitea's separation of generic Server and protocol
// details while retaining BDSPro's http.Server.Shutdown and handler drain.
func newHTTPServer(
	manager *Manager,
	network, address, name string,
	handler http.Handler,
	shutdownTimeout time.Duration,
	onListening func(net.Addr),
) (*Server, ServeFunction) {
	active := &activeHTTPHandlers{next: handler}
	httpServer := &http.Server{
		BaseContext:       func(net.Listener) context.Context { return manager.HammerContext() },
		Handler:           active,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    16 << 10,
	}
	srv := NewServer(manager, network, address, name)
	srv.ShutdownTimeout = shutdownTimeout
	srv.OnListening = onListening
	srv.OnShutdown = func(ctx context.Context) error {
		httpServer.SetKeepAlivesEnabled(false)
		return httpServer.Shutdown(ctx)
	}
	srv.OnHammer = httpServer.Close
	srv.OnDrained = active.Drain
	return srv, httpServer.Serve
}

// HTTPListenAndServe belongs to graceful, not cmd. It is the sole HTTP
// listen/drain entrypoint for BDSPro Web.
func HTTPListenAndServe(
	manager *Manager,
	network, address, name string,
	handler http.Handler,
	shutdownTimeout time.Duration,
	onListening func(net.Addr),
) error {
	srv, serve := newHTTPServer(manager, network, address, name, handler, shutdownTimeout, onListening)
	return srv.ListenAndServe(serve)
}
