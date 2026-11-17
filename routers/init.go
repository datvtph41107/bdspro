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
	"github.com/datvtph41107/bdspro/routers/common"
	"github.com/gin-gonic/gin"
)

func InitWebInstalled(ctx context.Context) (err error) {
	if err := setting.LoadSettings(); err != nil {
		return fmt.Errorf("load Web settings: %w", err)
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	// Database startup policy belongs to routers/common.
	if err := common.InitDBEngine(ctx); err != nil {
		return err
	}

	redisReady := false

	// At this point PostgreSQL has been initialized.
	// Roll back completed resources if a later step fails.
	defer func() {
		if err != nil {
			if redisReady {
				conn.CloseRedis()
			}
			db.CloseEngine()
		}
	}()

	log.Info("PostgreSQL ready")

	if err := ctx.Err(); err != nil {
		return err
	}

	// Redis startup policy is kept alongside database startup policy.
	// The Redis module still owns its client and graceful cleanup registration.
	if err := common.InitRedis(ctx); err != nil {
		return err
	}

	redisReady = true

	return ctx.Err()
}

// NormalRoutes registers the one API route tree. No domain routes are installed.
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
	router := routes.GinEngine()
	// Swagger serves net/http directly; this is a temporary Gin route
	// registration boundary until the parent matcher migrates to Chi.
	if swagger := api.SwaggerHandler(setting.Common.RunMode); swagger != nil {
		router.GET("/swagger/*filepath", func(c *gin.Context) {
			swagger.ServeHTTP(c.Writer, c.Request)
		})
	}
	if err := router.SetTrustedProxies(setting.Server.TrustedProxies); err != nil {
		return nil, fmt.Errorf("configure Gin trusted proxies: %w", err)
	}
	return routes.WrapIngress(), nil
}
