package api

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	db "github.com/datvtph41107/bdspro/db/sqlc"
	"github.com/datvtph41107/bdspro/token"
	"github.com/datvtph41107/bdspro/util"
	"github.com/gin-gonic/gin"
)

type Server struct {
	config     util.Config
	store      db.Store
	tokenMaker token.Maker
	router     *gin.Engine
}

func NewServer(config util.Config, store db.Store) (*Server, error) {
	maker, err := token.NewJWTMaker(config.TokenSymmetricKey)
	if err != nil {
		return nil, fmt.Errorf("create token maker: %w", err)
	}
	server := &Server{config: config, store: store, tokenMaker: maker}
	server.setupRouter()
	return server, nil
}

func (server *Server) setupRouter() {
	router := gin.New()
	router.Use(gin.Recovery(), requestLogger(), corsMiddleware(server.config.AllowedOrigins))
	router.GET("/health", func(ctx *gin.Context) {
		ctx.Status(http.StatusOK)
	})
	router.POST("/users", server.createUser)
	router.POST("/users/login", server.loginUser)
	router.POST("/tokens/renew_access", server.renewAccessToken)
	router.POST("/tokens/revoke", server.revokeRefreshToken)

	authorized := router.Group("/").Use(authMiddleware(server.tokenMaker))
	authorized.GET("/users/me", server.me)
	server.router = router
}

func (server *Server) me(ctx *gin.Context) {
	payload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
	ctx.JSON(http.StatusOK, gin.H{"account_id": payload.AccountID})
}

func (server *Server) Start(ctx context.Context, address string) error {
	httpServer := &http.Server{
		Addr:              address,
		Handler:           server.router,
		ReadHeaderTimeout: 5 * time.Second,
	}
	errorsCh := make(chan error, 1)
	go func() {
		errorsCh <- httpServer.ListenAndServe()
	}()

	select {
	case err := <-errorsCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown HTTP server: %w", err)
		}
		err := <-errorsCh
		if errors.Is(err, http.ErrServerClosed) {
			slog.Info("HTTP server stopped")
			return nil
		}
		return err
	}
}
