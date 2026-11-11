// Package routers initializes Web dependencies and composes HTTP routes.
package routers

import (
	"context"
	"os"

	"github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/modules/conn"
	"github.com/datvtph41107/bdspro/modules/graceful"
	"github.com/datvtph41107/bdspro/modules/log"
	"github.com/datvtph41107/bdspro/modules/setting"
	"github.com/datvtph41107/bdspro/modules/web"
	apiv1 "github.com/datvtph41107/bdspro/routers/api/v1"
	"github.com/datvtph41107/bdspro/routers/common"
	"github.com/datvtph41107/bdspro/services/mail"
	security "github.com/datvtph41107/bdspro/services/security"
	"github.com/gin-gonic/gin"
)

// InitWebInstalled owns the order of initialization for one Web process.
func InitWebInstalled(ctx context.Context) (*Server, error) {
	if err := setting.LoadSettings(); err != nil {
		return nil, err
	}
	lifecycle := graceful.GetManager()

	store, err := db.InitStore(ctx, setting.Database.DBSource)
	if err != nil {
		return nil, err
	}
	lifecycle.RunAtTerminate(store.Close)
	log.Info("PostgreSQL connected")

	// Development migrations remain opt-in by environment as before.
	if setting.Common.RunMode == "development" {
		changed, err := db.MigrateUp(setting.Database.MigrationURL, setting.Database.DBSource, 0)
		if err != nil {
			store.Close()
			return nil, err
		}
		if changed {
			log.Info("Migration status: applied")
		} else {
			log.Info("Migration status: no_change")
		}
	} else {
		log.Info("Database migrations are managed externally")
	}

	redisManager := conn.NewRedisManager()
	lifecycle.RunAtTerminate(func() {
		if err := redisManager.Close(); err != nil {
			log.Error("Close Redis: %v", err)
		}
	})
	redisClient, err := redisManager.Client(ctx, setting.Redis.Address)
	if err != nil {
		_ = redisManager.Close()
		store.Close()
		return nil, err
	}
	log.Info("Redis connected")

	if os.Getenv(gin.EnvGinMode) == "" {
		gin.SetMode(gin.ReleaseMode)
	}
	cfg := setting.WebConfig()
	sender := mail.NewSMTPSender(cfg.EmailSenderName, cfg.EmailSenderAddress,
		cfg.EmailSenderPassword, cfg.SMTPAddress, cfg.SMTPAuth)
	server, err := NewServer(cfg, store, redisClient, sender)
	if err != nil {
		_ = redisManager.Close()
		store.Close()
		return nil, err
	}

	return server, nil
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
