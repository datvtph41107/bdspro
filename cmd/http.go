package cmd

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/datvtph41107/bdspro/modules/log"
)

// serveHTTP owns the HTTP listener, readiness notification, and graceful
// shutdown. Router packages only provide net/http.Handler.
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

	httpServer := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    16 << 10,
	}
	errorsCh := make(chan error, 1)
	go func() {
		errorsCh <- httpServer.Serve(listener)
	}()
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
		shutdownErr := shutdownHTTPServer(httpServer, 10*time.Second)
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

// shutdownHTTPServer drains live HTTP requests until the deadline. On timeout
// it closes active connections before returning; no caller may close the DB
// while the server is still accepting connections. This cannot stop handlers
// that ignore canceled request contexts or hijacked connections.
func shutdownHTTPServer(server *http.Server, timeout time.Duration) error {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		// Shutdown does not interrupt active connections when its deadline
		// expires. Close forces them to disconnect; preserve the original
		// shutdown error so the CLI can report incomplete graceful drain.
		shutdownErr := fmt.Errorf("shutdown HTTP server: %w", err)
		if closeErr := server.Close(); closeErr != nil {
			return errors.Join(shutdownErr, fmt.Errorf("close HTTP server: %w", closeErr))
		}
		return shutdownErr
	}
	return nil
}
