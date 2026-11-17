// Package routers composes the HTTP infrastructure without domain services.
package routers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/modules/auth"
	"github.com/datvtph41107/bdspro/modules/conn"
	"github.com/datvtph41107/bdspro/modules/log"
	"github.com/datvtph41107/bdspro/modules/setting"
	"github.com/datvtph41107/bdspro/modules/web"
	api "github.com/datvtph41107/bdspro/routers/api"
	apiv1 "github.com/datvtph41107/bdspro/routers/api/v1"
	"github.com/datvtph41107/bdspro/routers/common"
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

	// The process JWT verifier is built from validated security settings.
	// Access tokens are never accepted without an active DB session.
	if err := auth.Init(); err != nil {
		return fmt.Errorf("initialize access token verifier: %w", err)
	}

	return ctx.Err()
}

// NormalRoutes registers the versioned API tree, including Organization/Team.
func NormalRoutes() *web.Router {
	r := web.NewRouter()
	r.Mount("/api/v1", apiv1.Routes())
	return r
}

func NormalHandler() (http.Handler, error) {
	routes := NormalRoutes()
	if swagger := api.SwaggerHandler(setting.Common.RunMode); swagger != nil {
		routes.Get("/swagger/*", swagger.ServeHTTP)
	}
	if err := routes.SetTrustedProxies(setting.Server.TrustedProxies); err != nil {
		return nil, fmt.Errorf("configure HTTP trusted proxies: %w", err)
	}
	return routes.WrapIngress(), nil
}
