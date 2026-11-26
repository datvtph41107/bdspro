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

	server.OnShutdown = func() {
		ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		if err := httpServer.Shutdown(ctx); err != nil {
			log.Printf("graceful: HTTP shutdown deadline: %v", err)

			if closeErr := httpServer.Close(); closeErr != nil {
				log.Printf("graceful: force close HTTP: %v", closeErr)
			}
		}
	}
	return server, httpServer.Serve, nil
}

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
