package server

import (
	"context"
	"errors"
	"fmt"
	apicontrollers "github.com/datvtph41107/bdspro/app/http/controllers/api"
	authcontrollers "github.com/datvtph41107/bdspro/app/http/controllers/auth"
	"github.com/datvtph41107/bdspro/app/services"
	"github.com/datvtph41107/bdspro/routes"
	"net"
	"net/http"
	"time"

	"github.com/datvtph41107/bdspro/auth"
	db "github.com/datvtph41107/bdspro/db/sqlc"
	"github.com/datvtph41107/bdspro/token"
	"github.com/datvtph41107/bdspro/util"
	"github.com/datvtph41107/bdspro/worker"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/rs/cors"
	"github.com/rs/zerolog/log"
)

type Server struct {
	authController         *authcontrollers.Controller
	organizationController *apicontrollers.OrganizationController
	config                 util.Config
	store                  db.Store
	verificationScheduler  auth.VerificationTaskScheduler
	tokenMaker             token.Maker
	router                 *gin.Engine
	authRateRedis          *redis.Client
	phoneOTP               db.PhoneOTPRepository
}

func NewServer(config util.Config, store db.Store, taskDistributor worker.TaskDistributor) (*Server, error) {
	maker, err := token.NewJWTMaker(config.TokenSymmetricKey)
	if err != nil {
		return nil, fmt.Errorf("create token maker: %w", err)
	}
	server := &Server{
		organizationController: apicontrollers.NewOrganizationController(services.NewOrganizationService(store)),
		config:                 config,
		store:                  store,
		tokenMaker:             maker,
		verificationScheduler:  worker.NewVerificationScheduler(taskDistributor),
	}
	server.authController = authcontrollers.NewController(store, server.verificationScheduler, maker, config)
	server.setupRouter()
	return server, nil
}

// EnableAuthRateLimiting must be called before Start. Redis outages reject auth requests.
func (server *Server) EnableAuthRateLimiting(client *redis.Client) error {
	if client == nil {
		return errors.New("authentication rate limiter requires Redis")
	}
	server.authRateRedis = client
	return nil
}

func (server *Server) setupRouter() {
	router := gin.New()
	// Direct Gin ingress: do not trust client-controlled forwarding headers.
	// If deploying behind a proxy, configure its exact trusted addresses explicitly.
	_ = router.SetTrustedProxies(nil)
	// Generate a server-owned ID for each request; do not trust a client-supplied ID.
	router.Use(func(ctx *gin.Context) {
		requestID := uuid.NewString()
		ctx.Header("X-Request-ID", requestID)
		requestLogger := log.With().Str("request_id", requestID).Logger()
		ctx.Request = ctx.Request.WithContext(requestLogger.WithContext(ctx.Request.Context()))
		ctx.Next()
	})
	// Never log the raw URL: email verification secrets can be in query parameters.
	router.Use(func(ctx *gin.Context) {
		startedAt := time.Now()
		ctx.Next()

		path := ctx.FullPath()
		if path == "" {
			// Unknown request paths can embed secrets; log a fixed label instead.
			path = "<unmatched>"
		}

		status := ctx.Writer.Status()
		event := log.Ctx(ctx.Request.Context()).Info()
		if status >= http.StatusInternalServerError {
			event = log.Ctx(ctx.Request.Context()).Error()
		}

		event.
			Str("method", ctx.Request.Method).
			Str("path", path).
			Int("status", status).
			Dur("duration", time.Since(startedAt)).
			Msg("http.request")
	},
		gin.CustomRecoveryWithWriter(nil, func(ctx *gin.Context, recovered any) {
			// Không ghi panic value hoặc raw URL vì có thể chứa secret.
			path := ctx.FullPath()
			if path == "" {
				path = "<unmatched>"
			}

			log.Ctx(ctx.Request.Context()).Error().
				Str("method", ctx.Request.Method).
				Str("path", path).
				Str("panic_type", fmt.Sprintf("%T", recovered)).
				Msg("http.panic")

			ctx.AbortWithStatus(http.StatusInternalServerError)
		}))

	// Global request boundary: observability wraps the policy and controller.
	// Admission and authentication run only on the routes that need them.
	router.Use(server.RequestMonitor(), RequestIngress())

	routes.Mount(router, routes.Handlers{
		Register: server.authController.Register, Login: server.authController.Login, VerifyEmail: server.authController.VerifyEmail,
		Renew: server.authController.Renew, Revoke: server.authController.Revoke, Me: server.me,
		CreateOrganization:    server.organizationController.Create,
		ListOrganizations:     server.organizationController.Index,
		AddOrganizationMember: server.organizationController.AddMember,
		UpdateOrganization:    server.organizationController.Update,
		PreRegister:           PreRegister(), PreLogin: PreLogin(), PreVerifyEmail: PreVerifyEmail(),
		PreToken: PreToken(), SufLogin: SufLogin(), Authenticate: Authenticate(server.tokenMaker),
		RegisterIdentity: registerIdentity, LoginIdentity: loginIdentity,
		VerifyEmailIdentity: emailVerificationIdentity, TokenIdentity: tokenIdentity,
		Source: server.IPActiveSource, Identity: server.IPActive,
	})
	server.router = router
}

func (server *Server) me(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{"account_id": authenticatedAccountID(ctx)})
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
	if (server.config.Environment == "development" || server.config.Environment == "production") && server.authRateRedis == nil {
		return errors.New("IPActive Redis limiter must be configured before starting HTTP")
	}
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
