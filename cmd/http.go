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
// Start runs the HTTP server until shutdown or a serving error.
// StartWithReady invokes onListening only after the TCP port has been bound.
func serveHTTP(ctx context.Context, handler http.Handler, address string, onListening func(net.Addr)) error {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("listen HTTP: %w", err)
	}
	defer listener.Close()

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
	log.Event(log.INFO).Str("address", listener.Addr().String()).Msg("http.listening")
	if onListening != nil {
		onListening(listener.Addr())
	}

	select {
	case err := <-errorsCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("serve HTTP: %w", err)
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown HTTP server: %w", err)
		}
		if err := <-errorsCh; err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve HTTP: %w", err)
		}
		log.Event(log.INFO).Msg("http.stopped")
		return nil
	}
}

