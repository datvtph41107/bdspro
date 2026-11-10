// Package routers builds the HTTP application using Gin and existing modules.
package routers

import (
	"os"

	"github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/modules/setting"
	"github.com/datvtph41107/bdspro/modules/web"
	"github.com/datvtph41107/bdspro/routers/api"
	apiv1 "github.com/datvtph41107/bdspro/routers/api/v1"
	"github.com/datvtph41107/bdspro/routers/common"
	appctx "github.com/datvtph41107/bdspro/services/context"
	"github.com/datvtph41107/bdspro/services/mail"
	security "github.com/datvtph41107/bdspro/services/security"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

// InitWebInstalled initializes Gin policy and route-bound dependencies.
// The existing routers/api package owns route declarations and HTTP contracts.
func InitWebInstalled(cfg setting.Config, store db.Store, redisClient *redis.Client) (*Server, error) {
	if os.Getenv(gin.EnvGinMode) == "" {
		gin.SetMode(gin.ReleaseMode)
	}
	sender := mail.NewSMTPSender(cfg.EmailSenderName, cfg.EmailSenderAddress,
		cfg.EmailSenderPassword, cfg.SMTPAddress, cfg.SMTPAuth)
	return NewServer(cfg, store, redisClient, sender)
}

// NormalRoutes registers the API and Swagger on one Gin matcher, then binds
// the HTTP BeforeRouting chain around that exact matcher. The resulting
// handler is the only production HTTP ingress; Server.Handler serves it.
func NormalRoutes(server *Server) *gin.Engine {
	// The parent web.Router owns the only production Gin matcher.
	r := web.NewRouter()
	router := r.GinEngine()

	router.Use(common.ProtocolMiddlewares()...)

	router.Use(
		security.RequestMonitor(server.authRateRedis),
		appctx.RequestIngress(),
	)
	// Install the immutable DB capability once per request. The nil guard
	// keeps public/registration-only test routers independent of a database.
	router.Use(func(c *gin.Context) {
		if server.store != nil {
			c.Request = c.Request.WithContext(db.WithStore(c.Request.Context(), server.store))
		}
		c.Next()
	})

	// Gin transport constructs APIContext after a route is resolved. The
	// mounted subrouter owns route declarations; the parent owns transport.
	router.Use(appctx.APIContexter())
	r.Mount("/", apiv1.MountedRoutes(apiv1.Dependencies{
		Auth:          server.authHandler,
		OAuth2:        server.oauth2Handler,
		Authorization: server.authorizationHandler,
		Browser:       server.browserHandler,
		PhoneEnabled:  server.phoneEnabled,
		TokenMaker:    server.tokenMaker,
		IPActive:      server.ipActive,
		Store:         server.store,
		InviteKey:     server.config.TokenSymmetricKey,
	}))
	api.RegisterSwagger(router, server.config.Environment)

	// Serve the single parent matcher behind its pre-routing HTTP ingress.
	server.httpHandler = r.WrapIngress()
	return router
}
