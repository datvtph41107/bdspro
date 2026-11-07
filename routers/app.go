package routers

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	db "github.com/datvtph41107/bdspro/models/db"
	authcrypto "github.com/datvtph41107/bdspro/modules/auth"
	"github.com/datvtph41107/bdspro/modules/ipactive"
	"github.com/datvtph41107/bdspro/modules/log"
	"github.com/datvtph41107/bdspro/modules/setting"
	"github.com/datvtph41107/bdspro/routers/api/middleware"
	"github.com/datvtph41107/bdspro/routers/api/response"
	"github.com/datvtph41107/bdspro/routers/api/v1/oauth"
	"github.com/datvtph41107/bdspro/routers/api/v1/org"
	"github.com/datvtph41107/bdspro/routers/api/v1/user"
	"github.com/datvtph41107/bdspro/routers/common"
	weboauth "github.com/datvtph41107/bdspro/routers/web/oauth"
	auth_service "github.com/datvtph41107/bdspro/services/auth"
	"github.com/datvtph41107/bdspro/services/mail"
	"github.com/datvtph41107/bdspro/services/notification"
	oauth_service "github.com/datvtph41107/bdspro/services/oauth"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

// Server owns initialized HTTP dependencies and the configured Gin application.
// Listener, readiness, timeouts and graceful shutdown remain in cmd/http.go.
type Server struct {
	authHandler          *user.AuthHandler
	organizationHandler  *org.Handler
	oauth2Handler        *user.OAuth2Handler
	authorizationHandler *oauth.Handler
	browserHandler       *weboauth.Handler
	phoneEnabled         bool
	config               setting.Config
	store                db.Store
	tokenMaker           authcrypto.Maker
	router               *gin.Engine
	authRateRedis        *redis.Client
	authThrottle         *middleware.AuthThrottle
	ipActive             *ipactive.Middleware
}

// NewServer constructs dependencies and registers every HTTP route exactly once.
func NewServer(cfg setting.Config, store db.Store, redisClient *redis.Client, resetMailers ...mail.EmailSender) (*Server, error) {
	if redisClient == nil {
		return nil, errors.New("authentication requires Redis")
	}
	if err := setting.ValidateTrustedProxies(cfg.TrustedProxies); err != nil {
		return nil, fmt.Errorf("configure trusted ingress peers: %w", err)
	}
	maker, err := authcrypto.NewJWTMaker(cfg.Authentication().JWTSecret)
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
	policies := cfg.IPActivePolicies
	if len(policies) == 0 {
		policies = ipactive.DefaultPolicies()
	}
	limiter, err := ipactive.NewRedis(redisClient, policies)
	if err != nil {
		return nil, fmt.Errorf("initialize IPActive: %w", err)
	}
	server.ipActive = ipactive.NewMiddleware(limiter, func(c *gin.Context, event ipactive.Event, admissionErr error) {
		if admissionErr != nil {
			log.Ctx(c.Request.Context()).Error().
				Str("action", string(event.Action)).
				Str("error_type", fmt.Sprintf("%T", admissionErr)).
				Msg("ipactive.admission_unavailable")
			response.RateLimitUnavailable(c)
			return
		}
		// Log the actual admission-decision time, not a later handler timestamp.
		// No raw client IP or Redis key is written to the application log.
		log.Ctx(c.Request.Context()).Warn().
			Str("action", string(event.Action)).
			Int64("occurred_at_unix", int64(event.OccurredAt)).
			Int64("attempts", event.Attempts).
			Msg("ipactive.rate_limited")
		response.RateLimited(c, event.RetryAfter)
	})
	// Construct a single authentication owner for all credential methods.
	phoneSender, err := notification.NewPhoneCodeSender(cfg)
	if err != nil {
		return nil, err
	}
	authService := auth_service.NewAuthService(store, maker, cfg, phoneSender)
	var resetSender mail.EmailSender
	if len(resetMailers) > 0 {
		resetSender = resetMailers[0]
	}
	server.authHandler = user.NewAuthHandler(authService, server.authThrottle, resetSender)
	server.phoneEnabled = phoneSender != nil
	server.organizationHandler = org.NewHandler(store)
	if oauthStore, ok := store.(db.OAuthServerStore); ok {
		proofs, err := oauth_service.NewConsentProofs(cfg.Authentication().JWTSecret)
		if err != nil {
			return nil, fmt.Errorf("configure OAuth consent proof: %w", err)
		}
		oauthService := oauth_service.NewService(oauthStore)
		server.authorizationHandler = oauth.NewHandler(oauthService, store, proofs)
		if cfg.BrowserPublicOrigin != "" {
			if err := setting.ValidateBrowserPublicOrigin(cfg.BrowserPublicOrigin, cfg.Environment); err != nil {
				return nil, err
			}
			codec, err := authcrypto.NewBrowserSessionCodec(cfg.TokenSymmetricKey)
			if err != nil {
				return nil, fmt.Errorf("configure browser session: %w", err)
			}
			server.browserHandler, err = weboauth.NewHandler(weboauth.Config{
				Store: store, OAuth: oauthService, API: server.authorizationHandler,
				Proofs: proofs, Login: authService.Login, Maker: maker, Codec: codec,
				PublicOrigin: cfg.BrowserPublicOrigin, Throttle: server.authThrottle,
			})
			if err != nil {
				return nil, fmt.Errorf("configure browser OAuth: %w", err)
			}
		}
	}

	// OIDC owns its browser protocol, but account linking and session
	// issuance converge on the same AuthService.
	if cfg.OIDC.Enabled() {
		client := &http.Client{Timeout: 10 * time.Second}
		oauth2, err := auth_service.NewOAuth2Client(cfg.OIDC, redisClient, client)
		if err != nil {
			return nil, fmt.Errorf("configure oauth2: %w", err)
		}
		server.oauth2Handler = user.NewOAuth2Handler(
			authService, oauth2, strings.HasPrefix(cfg.OIDC.RedirectURI, "https://"),
		)
	}

	server.router = NormalRoutes(server)
	if err := server.router.SetTrustedProxies(cfg.TrustedProxies); err != nil {
		return nil, fmt.Errorf("configure Gin trusted proxies: %w", err)
	}
	return server, nil
}

// Handler exposes Gin routes with the same CORS policy as the previous HTTP ingress.
func (server *Server) Handler() http.Handler {
	return common.CORS(server.router, server.config.AllowedOrigins)
}
