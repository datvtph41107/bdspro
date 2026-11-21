package graceful

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"
)

// newHTTPServer adapts net/http to the process Manager.
// HTTP Shutdown waits for active handlers before DB/Redis termination.
func newHTTPServer(
	manager *Manager,
	network, address, name string,
	handler http.Handler,
	shutdownTimeout time.Duration,
) (*Server, ServeFunction, error) {
	if manager == nil {
		return nil, nil, errors.New("graceful: nil HTTP manager")
	}
	if handler == nil {
		return nil, nil, errors.New("graceful: nil HTTP handler")
	}
	if shutdownTimeout <= 0 {
		return nil, nil, fmt.Errorf("graceful: invalid HTTP shutdown timeout %s", shutdownTimeout)
	}
	server := NewServer(manager, network, address, name)
	httpServer := http.Server{
		Handler: handler,
		BaseContext: func(net.Listener) context.Context {
			return manager.HammerContext()
		},
		ReadHeaderTimeout: 10 * time.Second,
	}
	// This hook runs exactly once under Server.shutdownOnce. It is invoked
	// from a separate lifecycle goroutine, not from the HTTP Serve loop.
	server.OnShutdown = func() {
		ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		// Shutdown closes listeners and idle connections, and waits for active
		// requests to become idle. It does not wait for hijacked connections.
		if err := httpServer.Shutdown(ctx); err != nil {
			log.Printf("graceful: HTTP shutdown deadline: %v", err)
			// When the deadline is exceeded, force-close active net/http conns.
			// This cannot promise that an uncooperative handler has exited.
			if closeErr := httpServer.Close(); closeErr != nil {
				log.Printf("graceful: force close HTTP: %v", closeErr)
			}
		}
	}
	return server, httpServer.Serve, nil
}

// HTTPListenAndServe owns the HTTP protocol adapter and delegates connection
// tracking/termination to Server and process shutdown to Manager.
func HTTPListenAndServe(
	network, address, name string,
	handler http.Handler,
	shutdownTimeout time.Duration,
) error {
	server, serve, err := newHTTPServer(
		GetManager(), network, address, name, handler, shutdownTimeout,
	)
	if err != nil {
		return err
	}
	return server.ListenAndServe(serve)
}
