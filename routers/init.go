// Package routers builds the HTTP application using Gin and existing modules.
package routers

import (
	"os"

	"github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/modules/setting"
	"github.com/datvtph41107/bdspro/modules/web"
	apiv1 "github.com/datvtph41107/bdspro/routers/api/v1"
	"github.com/datvtph41107/bdspro/routers/common"
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

// NormalRoutes has one route registration owner and no application singleton.
// Per-Server capabilities are added to the request before this router serves.
func NormalRoutes() *web.Router {
	r := web.NewRouter()
	router := r.GinEngine()
	router.Use(common.ProtocolMiddlewares()...)
	router.Use(security.RequestMonitor(), common.RequestIngress())
	r.Mount("/", apiv1.Routes())
	return r
}
