package cmd

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/datvtph41107/bdspro/modules/log"
	"github.com/datvtph41107/bdspro/modules/setting"
)

// activeHTTPHandlers tracks all requests already admitted by the server.
// It prevents database/Redis termination before a handler has actually
// returned, even when http.Server.Close interrupts client connections.
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

// serveHTTP owns listener readiness, serving, graceful shutdown and drain.
func serveHTTP(ctx context.Context, handler http.Handler, address string, onListening func(net.Addr)) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("start HTTP: %w", err)
	}
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("listen HTTP: %w", err)
	}
	defer listener.Close()
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("start HTTP: %w", err)
	}

	active := &activeHTTPHandlers{next: handler}
	defer active.Drain()

	httpServer := &http.Server{
		Handler:           active,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    16 << 10,
	}
	errorsCh := make(chan error, 1)
	go func() { errorsCh <- httpServer.Serve(listener) }()
	if ctx.Err() == nil {
		log.Event(log.INFO).Str("address", listener.Addr().String()).Msg("http.listening")
		if onListening != nil {
			onListening(listener.Addr())
		}
	}
	select {
	case err := <-errorsCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("serve HTTP: %w", err)
	case <-ctx.Done():
		shutdownErr := shutdownHTTPServer(httpServer, setting.HTTPShutdownTimeout())
		serveErr := <-errorsCh
		if serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			return errors.Join(shutdownErr, fmt.Errorf("serve HTTP: %w", serveErr))
		}
		if shutdownErr != nil {
			return shutdownErr
		}
		log.Event(log.INFO).Msg("http.stopped")
		return nil
	}
}

func shutdownHTTPServer(server *http.Server, timeout time.Duration) error {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		shutdownErr := fmt.Errorf("shutdown HTTP server: %w", err)
		if closeErr := server.Close(); closeErr != nil {
			return errors.Join(shutdownErr, fmt.Errorf("close HTTP server: %w", closeErr))
		}
		return shutdownErr
	}
	return nil
}
