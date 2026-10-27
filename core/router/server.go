package router

import (
    "context"
    "errors"
    "fmt"
    "net"
    "net/http"
    "net/url"
    "time"

    "github.com/datvtph41107/bdspro/api/handler"
    "github.com/datvtph41107/bdspro/api/service"
    "github.com/datvtph41107/bdspro/config"
    "github.com/datvtph41107/bdspro/core/middleware"
    db "github.com/datvtph41107/bdspro/db/sqlc"
    "github.com/datvtph41107/bdspro/routes"
    "github.com/datvtph41107/bdspro/token"
    "github.com/datvtph41107/bdspro/worker"
    "github.com/gin-gonic/gin"
    "github.com/go-redis/redis/v8"
    "github.com/google/uuid"
    "github.com/rs/cors"
    "github.com/rs/zerolog/log"
)

const apiPrefix = "" // Preserve existing public endpoints; versioning is a separate API contract decision.

type Server struct {
    authHandler         *handler.AuthHandler
    organizationHandler *handler.OrganizationHandler
    googleHandler       *handler.GoogleHandler
    phoneHandler        *handler.PhoneHandler
    config              config.Config
    store               db.Store
    tokenMaker          token.Maker
    router              *gin.Engine
    authRateRedis       *redis.Client
    identityLimiter     *middleware.IdentityLimiter
}

// NewServer constructs dependencies and registers every HTTP route exactly once.
func NewServer(cfg config.Config, store db.Store, taskDistributor worker.TaskDistributor, redisClient *redis.Client) (*Server, error) {
    if redisClient == nil {
        return nil, errors.New("authentication requires Redis")
    }
    maker, err := token.NewJWTMaker(cfg.Authentication().JWTSecret)
    if err != nil {
        return nil, fmt.Errorf("create token maker: %w", err)
    }
    server := &Server{
        config: cfg,
        store: store,
        tokenMaker: maker,
        authRateRedis: redisClient,
        identityLimiter: middleware.NewIdentityLimiter(redisClient),
    }
    scheduler := worker.NewVerificationScheduler(taskDistributor)
    server.authHandler = handler.NewAuthHandler(store, scheduler, maker, cfg, server.identityLimiter)
    server.organizationHandler = handler.NewOrganizationHandler(service.NewOrganizationService(store))

    if cfg.GoogleClientID != "" {
        redirect, err := url.Parse(cfg.GoogleRedirectURI)
        if err != nil {
            return nil, fmt.Errorf("parse Google callback URL: %w", err)
        }
        server.googleHandler = handler.NewGoogleHandler(
            store, maker, cfg, redisClient,
            &http.Client{Timeout: 10 * time.Second}, redirect.Scheme == "https",
        )
    }
    if cfg.Environment == "development" {
        server.phoneHandler = handler.NewPhoneHandler(
            store, store, maker, cfg, server.identityLimiter,
        )
    }

    server.setupRouter()
    return server, nil
}

func (server *Server) setupRouter() {
    router := gin.New()
    // Forwarded headers are untrusted unless the deployment defines exact proxy addresses.
    _ = router.SetTrustedProxies(nil)

    router.Use(func(ctx *gin.Context) {
        requestID := uuid.NewString()
        ctx.Header("X-Request-ID", requestID)
        requestLogger := log.With().Str("request_id", requestID).Logger()
        ctx.Request = ctx.Request.WithContext(requestLogger.WithContext(ctx.Request.Context()))
        ctx.Next()
    })
    // Never log a raw URL: email verification tokens may appear in query strings.
    router.Use(func(ctx *gin.Context) {
        startedAt := time.Now()
        ctx.Next()

        path := ctx.FullPath()
        if path == "" {
            path = "<unmatched>"
        }
        status := ctx.Writer.Status()
        event := log.Ctx(ctx.Request.Context()).Info()
        if status >= http.StatusInternalServerError {
            event = log.Ctx(ctx.Request.Context()).Error()
        }
        event.Str("method", ctx.Request.Method).
            Str("path", path).
            Int("status", status).
            Dur("duration", time.Since(startedAt)).
            Msg("http.request")
    }, gin.CustomRecoveryWithWriter(nil, func(ctx *gin.Context, recovered any) {
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

    router.Use(
        middleware.RequestMonitor(server.authRateRedis),
        middleware.RequestIngress(),
    )

    routes.API(router, router.Group(apiPrefix), routes.Handlers{
        Auth: server.authHandler,
        Organization: server.organizationHandler,
        Google: server.googleHandler,
        Phone: server.phoneHandler,
    }, routes.Policies{
        Authenticate: middleware.Authenticate(server.tokenMaker),
        LoginAudit: middleware.LoginAudit(),
        SourceLimit: func(operation string, attempts int64, window time.Duration) gin.HandlerFunc {
            return middleware.IPActiveSource(server.authRateRedis, operation, attempts, window)
        },
    })

    server.router = router
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
