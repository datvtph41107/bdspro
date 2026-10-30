package api

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/datvtph41107/bdspro/internal/api/handler"
	"github.com/datvtph41107/bdspro/internal/api/middleware"
	"github.com/datvtph41107/bdspro/internal/api/response"
	"github.com/datvtph41107/bdspro/internal/config"
	db "github.com/datvtph41107/bdspro/internal/database"
	"github.com/datvtph41107/bdspro/internal/mail"
	"github.com/datvtph41107/bdspro/internal/notification"
	"github.com/datvtph41107/bdspro/internal/service"
	"github.com/datvtph41107/bdspro/internal/token"
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
	authThrottle        *middleware.AuthThrottle
}

// NewServer constructs dependencies and registers every HTTP route exactly once.
func NewServer(cfg config.Config, store db.Store, redisClient *redis.Client, resetMailers ...mail.EmailSender) (*Server, error) {
	if redisClient == nil {
		return nil, errors.New("authentication requires Redis")
	}
	maker, err := token.NewJWTMaker(cfg.Authentication().JWTSecret)
	if err != nil {
		return nil, fmt.Errorf("create token maker: %w", err)
	}
	server := &Server{
		config:        cfg,
		store:         store,
		tokenMaker:    maker,
		authRateRedis: redisClient,
		authThrottle:  middleware.NewAuthThrottle(redisClient),
	}
	server.authHandler = handler.NewAuthHandler(service.NewAuthService(store, maker, cfg), server.authThrottle)
	if len(resetMailers) > 0 {
		server.authHandler.SetPasswordResetSender(resetMailers[0])
	}
	server.organizationHandler = handler.NewOrganizationHandler(service.NewOrganizationService(store))

	if cfg.GoogleClientID != "" {
		redirect, err := url.Parse(cfg.GoogleRedirectURI)
		if err != nil {
			return nil, fmt.Errorf("parse Google callback URL: %w", err)
		}
		server.googleHandler = handler.NewGoogleHandler(
			service.NewGoogleAuthService(store, maker, cfg), cfg, redisClient,
			&http.Client{Timeout: 10 * time.Second}, redirect.Scheme == "https",
		)
	}
	// Local delivery is only enabled in development. Production phone binding
	// is unavailable until an explicitly configured real provider is supplied.
	var phoneSender notification.PhoneCodeSender
	if cfg.Environment == "development" {
		phoneSender = notification.Local{}
	} else if cfg.PhoneDelivery == "sms" {
		endpoint, parseErr := url.Parse(cfg.SMSGatewayURL)
		if parseErr != nil || endpoint == nil || endpoint.Scheme != "https" || endpoint.Host == "" || cfg.SMSGatewayToken == "" {
			return nil, errors.New("SMS_GATEWAY_URL must use HTTPS and SMS_GATEWAY_TOKEN is required")
		}
		phoneSender = notification.SMSGateway{
			URL: cfg.SMSGatewayURL, Token: cfg.SMSGatewayToken,
			Client: &http.Client{Timeout: 8 * time.Second},
		}
	} else if cfg.PhoneDelivery == "zalo" {
		if cfg.ZaloAccessToken == "" || cfg.ZaloTemplateID == "" || cfg.ZaloCodeField == "" {
			return nil, errors.New("Zalo ZBS delivery requires access token, template ID and code field")
		}
		phoneSender = notification.ZaloZBS{
			AccessToken: cfg.ZaloAccessToken, TemplateID: cfg.ZaloTemplateID,
			CodeField: cfg.ZaloCodeField, Client: &http.Client{Timeout: 8 * time.Second},
		}
	} else if cfg.PhoneDelivery != "" && cfg.PhoneDelivery != "disabled" {
		return nil, fmt.Errorf("unsupported PHONE_DELIVERY: %s", cfg.PhoneDelivery)
	}
	if phoneSender != nil {
		phone := service.PhoneOTP{Store: store, Config: cfg, Sender: phoneSender}
		server.phoneHandler = handler.NewPhoneHandler(
			store, cfg, phoneSender, server.authThrottle,
		)
		server.phoneHandler.SetTokenMaker(maker)
		if server.googleHandler != nil {
			server.googleHandler.SetPhoneBinding(phone)
		}
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
	router.Use(accessLogger(), gin.CustomRecoveryWithWriter(nil, func(ctx *gin.Context, recovered any) {
		path := ctx.FullPath()
		if path == "" {
			path = "<unmatched>"
		}
		log.Ctx(ctx.Request.Context()).Error().
			Str("method", ctx.Request.Method).
			Str("path", path).
			Str("panic_type", fmt.Sprintf("%T", recovered)).
			Msg("http.panic")
		response.InternalError(ctx)
	}))

	router.Use(
		middleware.RequestMonitor(server.authRateRedis),
		middleware.RequestIngress(),
	)

	server.registerRoutes(router)
	server.registerSwagger(router)

	server.router = router
}

// accessLogger records one safe request outcome, distinct from a failure
// diagnostic. Route templates avoid token-bearing query strings and raw IDs.
func accessLogger() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		startedAt := time.Now()
		ctx.Next()

		path := ctx.FullPath()
		if path == "" {
			path = "<unmatched>"
		}
		event := log.Ctx(ctx.Request.Context()).Info()
		if code := ctx.GetString(response.ErrorCodeKey); code != "" {
			event = event.Str("error_code", code)
		}
		event.Str("method", ctx.Request.Method).
			Str("path", path).
			Int("status", ctx.Writer.Status()).
			Dur("duration", time.Since(startedAt)).
			Msg("http.request")
	}
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
