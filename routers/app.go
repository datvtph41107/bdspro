package routers

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/modules/setting"
	"github.com/datvtph41107/bdspro/modules/token"
	"github.com/datvtph41107/bdspro/routers/api/middleware"
	"github.com/datvtph41107/bdspro/routers/api/v1/org"
	"github.com/datvtph41107/bdspro/routers/api/v1/user"
	"github.com/datvtph41107/bdspro/routers/common"
	auth_service "github.com/datvtph41107/bdspro/services/auth"
	"github.com/datvtph41107/bdspro/services/mail"
	"github.com/datvtph41107/bdspro/services/notification"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

// Server owns initialized HTTP dependencies and the configured Gin application.
// Listener, readiness, timeouts and graceful shutdown remain in cmd/http.go.
type Server struct {
	authHandler         *user.AuthHandler
	organizationHandler *org.Handler
	googleHandler       *user.GoogleHandler
	oauth2Handler       *user.OAuth2Handler
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
	if cfg.OIDC.Enabled() && cfg.GoogleClientID != "" {
		return nil, errors.New("OAuth2/OIDC and legacy OAuth provider cannot be enabled together")
	}
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
	if cfg.OIDC.Enabled() {
		client := &http.Client{Timeout: 10 * time.Second}
		flow, err := auth_service.NewOAuth2Flow(cfg.OIDC, redisClient, client)
		if err != nil {
			return nil, fmt.Errorf("configure oauth2: %w", err)
		}
		server.oauth2Handler = user.NewOAuth2Handler(
			auth_service.NewOAuth2AuthService(store, maker, cfg), flow,
			strings.HasPrefix(cfg.OIDC.RedirectURI, "https://"),
		)
	}
	phoneSender, err := notification.NewPhoneCodeSender(cfg)
	if err != nil {
		return nil, err
	}
	if phoneSender != nil {
		phone := auth_service.PhoneOTP{Store: store, Config: cfg, Sender: phoneSender}
		server.phoneHandler = user.NewPhoneHandler(
			store, cfg, phoneSender, server.authThrottle,
		)
		server.phoneHandler.SetTokenMaker(maker)
		if server.oauth2Handler != nil {
			server.oauth2Handler.SetPhoneBinding(phone)
		}
		if server.googleHandler != nil {
			server.googleHandler.SetPhoneBinding(phone)
		}
	}

	server.router = NormalRoutes(server)
	return server, nil
}

// Handler exposes Gin routes with the same CORS policy as the previous HTTP ingress.
func (server *Server) Handler() http.Handler {
	return common.CORS(server.router, server.config.AllowedOrigins)
}
