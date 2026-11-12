// Package routers composes the HTTP infrastructure without domain services.
package routers

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/modules/conn"
	"github.com/datvtph41107/bdspro/modules/log"
	"github.com/datvtph41107/bdspro/modules/setting"
	"github.com/datvtph41107/bdspro/modules/web"
	api "github.com/datvtph41107/bdspro/routers/api"
	apiv1 "github.com/datvtph41107/bdspro/routers/api/v1"
	"github.com/gin-gonic/gin"
)

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
	return nil
}

func NormalRoutes() *web.Router {
	r := web.NewRouter()
	r.Mount("/api/v1", apiv1.Routes())
	return r
}

func NormalHandler() (http.Handler, error) {
	if os.Getenv(gin.EnvGinMode) == "" {
		gin.SetMode(gin.ReleaseMode)
	}
	routes := NormalRoutes()
	api.RegisterSwagger(routes, setting.Common.RunMode)
	if err := routes.SetTrustedProxies(setting.Server.TrustedProxies); err != nil {
		return nil, fmt.Errorf("configure trusted proxies: %w", err)
	}
	// Registration errors must fail startup, never the first live request.
	routes.Compile()
	return routes, nil
}
