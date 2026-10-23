package api

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	db "github.com/datvtph41107/bdspro/db/sqlc"
	"github.com/datvtph41107/bdspro/token"
	"github.com/datvtph41107/bdspro/util"
	"github.com/datvtph41107/bdspro/worker"
	"github.com/gin-gonic/gin"
	"github.com/rs/cors"
	"github.com/rs/zerolog/log"
)

type Server struct {
	config          util.Config
	store           db.Store
	taskDistributor worker.TaskDistributor
	tokenMaker      token.Maker
	router          *gin.Engine
}

func NewServer(config util.Config, store db.Store, taskDistributor worker.TaskDistributor) (*Server, error) {
	maker, err := token.NewJWTMaker(config.TokenSymmetricKey)
	if err != nil {
		return nil, fmt.Errorf("create token maker: %w", err)
	}
	server := &Server{config: config, store: store, tokenMaker: maker, taskDistributor: taskDistributor}
	server.setupRouter()
	return server, nil
}

func (server *Server) setupRouter() {
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())
	router.GET("/health", func(ctx *gin.Context) {
		ctx.Status(http.StatusOK)
	})
	router.POST("/users", server.createUser)
	router.POST("/users/login", server.loginUser)
	router.GET("/users/verify_email", server.verifyEmail)
	router.POST("/tokens/renew_access", server.renewAccessToken)
	router.POST("/tokens/revoke", server.revokeRefreshToken)

	authorized := router.Group("/").Use(authMiddleware(server.tokenMaker))
	authorized.GET("/users/me", server.me)
	authorized.POST("/organizations", server.createOrganization)
	authorized.GET("/organizations", server.listOrganizations)
	authorized.POST("/organizations/:id/members", server.addOrganizationMember)
	authorized.PATCH("/organizations/:id", server.updateOrganization)
	server.router = router
}

func (server *Server) me(ctx *gin.Context) {
	payload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
	ctx.JSON(http.StatusOK, gin.H{"account_id": payload.AccountID})
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
