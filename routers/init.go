// Package routers orchestrates subsystem initialization and HTTP route composition.
package routers

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/datvtph41107/bdspro/models/db"
	authcrypto "github.com/datvtph41107/bdspro/modules/auth"
	"github.com/datvtph41107/bdspro/modules/conn"
	"github.com/datvtph41107/bdspro/modules/ipactive"
	"github.com/datvtph41107/bdspro/modules/log"
	"github.com/datvtph41107/bdspro/modules/setting"
	"github.com/datvtph41107/bdspro/modules/web"
	api "github.com/datvtph41107/bdspro/routers/api"
	apiv1 "github.com/datvtph41107/bdspro/routers/api/v1"
	"github.com/datvtph41107/bdspro/routers/common"
	weboauth "github.com/datvtph41107/bdspro/routers/web/oauth"
	auth_service "github.com/datvtph41107/bdspro/services/auth"
	oauth_service "github.com/datvtph41107/bdspro/services/oauth"
	org_service "github.com/datvtph41107/bdspro/services/org"
	security "github.com/datvtph41107/bdspro/services/security"
	"github.com/gin-gonic/gin"
)

// InitWebInstalled initializes package-owned subsystems in dependency order.
// Resource cleanup is owned by DB and Redis, registered with graceful.
func InitWebInstalled(ctx context.Context) error {
	if err := setting.LoadSettings(); err != nil {
		return err
	}
	if err := db.InitEngine(ctx); err != nil {
		return fmt.Errorf("initialize PostgreSQL: %w", err)
	}
	log.Info("PostgreSQL connected")
	if setting.Common.RunMode == "development" {
		changed, err := db.MigrateUp(setting.Database.MigrationURL, setting.Database.DBSource, 0)
		if err != nil {
			return err
		}
		if changed {
			log.Info("Migration status: applied")
		} else {
			log.Info("Migration status: no_change")
		}
	} else {
		log.Info("Database migrations are managed externally")
	}
	if _, err := conn.InitRedis(ctx); err != nil {
		return fmt.Errorf("initialize Redis: %w", err)
	}
	log.Info("Redis connected")
	if err := authcrypto.Init(); err != nil {
		return err
	}
	if err := security.Init(); err != nil {
		return err
	}
	if err := auth_service.Init(); err != nil {
		return err
	}
	if err := oauth_service.Init(); err != nil {
		return err
	}
	if err := weboauth.Init(); err != nil {
		return err
	}
	return nil
}

// NormalRoutes is the single route-registration owner.
func NormalRoutes() *web.Router {
	r := web.NewRouter()
	r.BeforeRouting(common.ProtocolMiddlewares()...)
	router := r.GinEngine()
	router.Use(common.AccessLogger(), common.Recovery(nil),
		security.RequestMonitor(), common.RequestIngress())
	r.Mount("/", apiv1.Routes())
	return r
}

// NormalHandler assembles HTTP routing using capabilities initialized by
// their owning packages. It never constructs a Store, AuthService, OAuth
// server, browser session, or Redis manager.
func NormalHandler() (http.Handler, error) {
	if os.Getenv(gin.EnvGinMode) == "" {
		gin.SetMode(gin.ReleaseMode)
	}
	routes := NormalRoutes()
	ginRouter := routes.GinEngine()
	api.RegisterSwagger(ginRouter, setting.Common.RunMode)
	if err := ginRouter.SetTrustedProxies(setting.Server.TrustedProxies); err != nil {
		return nil, fmt.Errorf("configure Gin trusted proxies: %w", err)
	}
	ingress := routes.WrapIngress()
	// Resolve process-owned capabilities once during startup, not on
	// every request. Runtime state is immutable after publication.
	authService := auth_service.ProcessService()
	oauthService, proofs := oauth_service.ProcessService()
	oidc, secure := auth_service.ProcessOAuth2()
	browser := weboauth.ProcessBrowser()
	redisClient, throttle, limiter := security.ProcessCapabilities()
	maker := authcrypto.ProcessMaker()
	signingKey := setting.Security.TokenSymmetricKey
	origins := append([]string(nil), setting.Server.AllowedOrigins...)
	return common.CORS(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		ctx = auth_service.WithService(ctx, authService)
		ctx = oauth_service.WithService(ctx, oauthService, proofs)
		ctx = auth_service.WithOAuth2(ctx, oidc, secure)
		ctx = weboauth.WithBrowser(ctx, browser)
		ctx = security.WithThrottle(ctx, throttle)
		ctx = security.WithMonitorRedis(ctx, redisClient)
		ctx = authcrypto.WithMaker(ctx, maker)
		ctx = ipactive.WithMiddleware(ctx, limiter)
		ctx = org_service.WithInviteSigningKey(ctx, signingKey)
		ingress.ServeHTTP(w, req.WithContext(ctx))
	}), origins), nil
}
