package routers

// This per-instance constructor is a test fixture only. Production lifecycle
// uses package-owned Init methods and routers.NormalHandler.

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	db "github.com/datvtph41107/bdspro/models/db"
	authcrypto "github.com/datvtph41107/bdspro/modules/auth"
	"github.com/datvtph41107/bdspro/modules/ipactive"
	"github.com/datvtph41107/bdspro/modules/setting"
	api "github.com/datvtph41107/bdspro/routers/api"
	"github.com/datvtph41107/bdspro/routers/common"
	weboauth "github.com/datvtph41107/bdspro/routers/web/oauth"
	auth_service "github.com/datvtph41107/bdspro/services/auth"
	"github.com/datvtph41107/bdspro/services/mail"
	"github.com/datvtph41107/bdspro/services/notification"
	oauth_service "github.com/datvtph41107/bdspro/services/oauth"
	org_service "github.com/datvtph41107/bdspro/services/org"
	security "github.com/datvtph41107/bdspro/services/security"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

// serverFixture is a test-only HTTP fixture that owns isolated dependencies and the one configured Gin matcher.
// Its httpHandler is the canonical pre-matcher ingress. Listener, readiness,
// timeouts and graceful shutdown remain in cmd/http.go.
type serverFixture struct {
	authService   *auth_service.AuthService
	oidcClient    *auth_service.OAuth2Client
	oidcSecure    bool
	oauthService  *oauth_service.Service
	oauthProofs   *oauth_service.ConsentProofs
	browser       *weboauth.Browser
	config        setting.Config
	store         db.Store
	tokenMaker    authcrypto.Maker
	router        *gin.Engine
	httpHandler   http.Handler
	authRateRedis *redis.Client
	authThrottle  *security.AuthThrottle
	ipActive      *ipactive.Middleware
}

// newServerFixture constructs dependencies and registers every HTTP route exactly once.
func newServerFixture(cfg setting.Config, store db.Store, redisClient *redis.Client, resetMailers ...mail.EmailSender) (*serverFixture, error) {
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
	server := &serverFixture{
		config:        cfg,
		store:         store,
		tokenMaker:    maker,
		authRateRedis: redisClient,
		authThrottle:  security.NewAuthThrottle(redisClient),
	}
	policies := cfg.IPActivePolicies
	if len(policies) == 0 {
		policies = ipactive.DefaultPolicies()
	}
	limiter, err := ipactive.NewRedis(redisClient, policies)
	if err != nil {
		return nil, fmt.Errorf("initialize IPActive: %w", err)
	}
	server.ipActive = ipactive.NewMiddleware(limiter)
	// Construct a single authentication owner for all credential methods.
	phoneSender, err := notification.NewPhoneCodeSender(cfg)
	if err != nil {
		return nil, err
	}
	var resetSender mail.EmailSender
	if len(resetMailers) > 0 {
		resetSender = resetMailers[0]
	}
	authService := auth_service.NewAuthService(store, maker, cfg, phoneSender, resetSender)
	server.authService = authService
	// Provider callback and browser origin must coincide before offering the
	// browser OIDC option. Mismatched deployments retain their existing API OIDC
	// callback and password-based browser consent without a misleading link.
	browserOIDC := cfg.OIDC.Enabled() && cfg.BrowserPublicOrigin != "" &&
		cfg.OIDC.RedirectURI == cfg.BrowserPublicOrigin+"/users/oauth2/callback"
	if oauthStore, ok := store.(db.OAuthServerStore); ok {
		proofs, err := oauth_service.NewConsentProofs(cfg.Authentication().JWTSecret)
		if err != nil {
			return nil, fmt.Errorf("configure OAuth consent proof: %w", err)
		}
		oauthService := oauth_service.NewService(oauthStore)
		server.oauthService = oauthService
		server.oauthProofs = proofs
		if cfg.BrowserPublicOrigin != "" {
			if err := setting.ValidateBrowserPublicOrigin(cfg.BrowserPublicOrigin, cfg.Environment); err != nil {
				return nil, err
			}
			codec, err := authcrypto.NewBrowserSessionCodec(cfg.TokenSymmetricKey)
			if err != nil {
				return nil, fmt.Errorf("configure browser session: %w", err)
			}
			server.browser, err = weboauth.NewBrowser(weboauth.Config{
				Codec: codec, PublicOrigin: cfg.BrowserPublicOrigin,
				OIDCEnabled: browserOIDC,
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
		server.oidcClient = oauth2
		server.oidcSecure = strings.HasPrefix(cfg.OIDC.RedirectURI, "https://")
	}

	routes := NormalRoutes()
	server.router = routes.GinEngine()
	api.RegisterSwagger(server.router, cfg.Environment)
	if err := server.router.SetTrustedProxies(cfg.TrustedProxies); err != nil {
		return nil, fmt.Errorf("configure Gin trusted proxies: %w", err)
	}
	// Complete the ingress chain during startup, before HTTP readiness.
	ingress := routes.WrapIngress()
	// Capabilities are instance-owned, not process-global route dependencies.
	server.httpHandler = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		ctx = auth_service.WithService(ctx, server.authService)
		ctx = oauth_service.WithService(ctx, server.oauthService, server.oauthProofs)
		ctx = auth_service.WithOAuth2(ctx, server.oidcClient, server.oidcSecure)
		ctx = weboauth.WithBrowser(ctx, server.browser)
		ctx = security.WithThrottle(ctx, server.authThrottle)
		ctx = security.WithMonitorRedis(ctx, server.authRateRedis)
		ctx = authcrypto.WithMaker(ctx, server.tokenMaker)
		ctx = ipactive.WithMiddleware(ctx, server.ipActive)
		ctx = org_service.WithInviteSigningKey(ctx, server.config.TokenSymmetricKey)
		if server.store != nil {
			ctx = db.WithStore(ctx, server.store)
		}
		ingress.ServeHTTP(w, req.WithContext(ctx))
	})
	return server, nil
}

// Handler serves the one composed HTTP ingress. CORS remains the outer
// protocol boundary and WrapIngress executes BeforeRouting before Gin matching.
// Never bypass this handler by serving server.router directly.
func (server *serverFixture) Handler() http.Handler {
	if server.httpHandler == nil {
		panic("routers: HTTP ingress was not composed")
	}
	return common.CORS(server.httpHandler, server.config.AllowedOrigins)
}
