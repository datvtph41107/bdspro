package api

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/modules/log"
	"github.com/datvtph41107/bdspro/modules/setting"
	"github.com/datvtph41107/bdspro/modules/token"
	"github.com/datvtph41107/bdspro/routers/api/middleware"
	"github.com/datvtph41107/bdspro/routers/api/response"
	apiv1 "github.com/datvtph41107/bdspro/routers/api/v1"
	"github.com/datvtph41107/bdspro/routers/api/v1/org"
	"github.com/datvtph41107/bdspro/routers/api/v1/user"
	"github.com/datvtph41107/bdspro/routers/common"
	auth_service "github.com/datvtph41107/bdspro/services/auth"
	"github.com/datvtph41107/bdspro/services/mail"
	"github.com/datvtph41107/bdspro/services/notification"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

const apiPrefix = "" // Preserve existing public endpoints; versioning is a separate API contract decision.

type Server struct {
	authHandler         *user.AuthHandler
	organizationHandler *org.Handler
	googleHandler       *user.GoogleHandler
	phoneHandler        *user.PhoneHandler
	config              setting.Config
	store               db.Store
	tokenMaker          token.Maker
	router              *gin.Engine
	authRateRedis       *redis.Client
	authThrottle        *middleware.AuthThrottle
}

// NewServer constructs dependencies and registers every HTTP route exactly once.
func NewServer(cfg setting.Config, store db.Store, redisClient *redis.Client, resetMailers ...mail.EmailSender) (*Server, error) {
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
	server.authHandler = user.NewAuthHandler(auth_service.NewAuthService(store, maker, cfg), server.authThrottle)
	if len(resetMailers) > 0 {
		server.authHandler.SetPasswordResetSender(resetMailers[0])
	}
	server.organizationHandler = org.NewHandler(store)

	if cfg.GoogleClientID != "" {
		redirect, err := url.Parse(cfg.GoogleRedirectURI)
		if err != nil {
			return nil, fmt.Errorf("parse Google callback URL: %w", err)
		}
		server.googleHandler = user.NewGoogleHandler(
			auth_service.NewGoogleAuthService(store, maker, cfg), cfg, redisClient,
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
		phone := auth_service.PhoneOTP{Store: store, Config: cfg, Sender: phoneSender}
		server.phoneHandler = user.NewPhoneHandler(
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

	router.Use(common.RequestID())
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

	apiv1.Routes(router, apiv1.Dependencies{
		Auth:         server.authHandler,
		Organization: server.organizationHandler,
		Google:       server.googleHandler,
		Phone:        server.phoneHandler,
		TokenMaker:   server.tokenMaker,
		RateRedis:    server.authRateRedis,
		Store:        server.store,
	})
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
	return common.CORS(server.router, server.config.AllowedOrigins)
}

