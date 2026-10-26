package server

import (
    "context"
    "errors"
    "fmt"
    "net"
    "net/http"
    "time"

    "github.com/datvtph41107/bdspro/util"
    "github.com/gin-gonic/gin"
    "github.com/rs/cors"
    "github.com/rs/zerolog/log"
)

// Server owns the HTTP listener and CORS handler. Route declarations and
// controllers are independent packages.
type Server struct {
    config util.Config
    router *gin.Engine
}

// Handler exposes Gin routes with the same CORS policy as the previous HTTP ingress.
func (server *Server) Handler() http.Handler {
	return cors.New(cors.Options{
		AllowedOrigins: server.config.AllowedOrigins,
		AllowedMethods: []string{
			http.MethodHead, http.MethodOptions, http.MethodGet, http.MethodPost,
			http.MethodPut, http.MethodPatch, http.MethodDelete,
		},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		ExposedHeaders:   []string{"X-Request-ID"},
		AllowCredentials: true,
	}).Handler(server.router)
}

// Start runs the HTTP server until shutdown or a serving error.
func (server *Server) Start(ctx context.Context, address string) error {
	return server.StartWithReady(ctx, address, nil)
}

// StartWithReady invokes onListening only after the TCP port has been bound.
func (server *Server) StartWithReady(ctx context.Context, address string, onListening func(net.Addr)) error {
		listener, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("listen HTTP: %w", err)
	}
	defer listener.Close()

	httpServer := &http.Server{
		Handler:           server.Handler(),
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
	log.Info().Str("address", listener.Addr().String()).Msg("http.listening")
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
		log.Info().Msg("http.stopped")
		return nil
	}
}
