// Package routers builds the HTTP application using Gin and existing modules.
package routers

import (
	"os"
	"sync"

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

// Router composition follows the Gitea pattern: NormalRoutes has no
// dependency arguments, and API routes are mounted as an independent Router.
// Dependencies are installed once in InitRouterRuntime during startup.
var (
	routerRuntimeMu sync.Mutex
	routerRuntime   *RouteRuntime
)

type RouteRuntime struct {
	Store         db.Store
	AuthRateRedis *redis.Client
	Environment   string
}

func InitRouterRuntime(config RouteRuntime) {
	routerRuntimeMu.Lock()
	defer routerRuntimeMu.Unlock()
	if config.AuthRateRedis == nil || routerRuntime != nil {
		panic("routers: invalid or duplicate router initialization")
	}
	copy := config
	routerRuntime = &copy
}

func NormalRoutes() *web.Router {
	routerRuntimeMu.Lock()
	config := routerRuntime
	routerRuntimeMu.Unlock()
	if config == nil {
		panic("routers: runtime not initialized")
	}
	r := web.NewRouter()
	router := r.GinEngine()

	router.Use(common.ProtocolMiddlewares()...)

	router.Use(
		security.RequestMonitor(config.AuthRateRedis),
		appctx.RequestIngress(),
	)
	// Install the immutable DB capability once per request. The nil guard
	// keeps public/registration-only test routers independent of a database.
	router.Use(func(c *gin.Context) {
		if config.Store != nil {
			c.Request = c.Request.WithContext(db.WithStore(c.Request.Context(), config.Store))
		}
		c.Next()
	})

	// Until the remaining Gin protocol actions become native HTTP actions,
	// APIContext is installed once by the parent matcher before typed actions.
	router.Use(appctx.APIContexter())
	r.Mount("/", apiv1.Routes())
	api.RegisterSwagger(router, config.Environment)
	return r
}
