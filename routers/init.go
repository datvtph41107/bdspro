// Package routers composes the HTTP infrastructure without domain services.
package routers

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/modules/auth"
	"github.com/datvtph41107/bdspro/modules/conn"
	"github.com/datvtph41107/bdspro/modules/ipactive"
	"github.com/datvtph41107/bdspro/modules/log"
	"github.com/datvtph41107/bdspro/modules/setting"
	"github.com/datvtph41107/bdspro/modules/startup"
	"github.com/datvtph41107/bdspro/modules/web"
	api "github.com/datvtph41107/bdspro/routers/api"
	apiv1 "github.com/datvtph41107/bdspro/routers/api/v1"
	"github.com/datvtph41107/bdspro/routers/common"
)

// InitWebInstalled initializes required Web dependencies in order. Resources
// retain their own graceful termination hooks; failed startup rolls back
// completed stages through idempotent resource-owner close functions.
func InitWebInstalled(ctx context.Context) error {
	return startup.Run(ctx,
		startup.Step{
			Name: "PostgreSQL",
			Init: func(ctx context.Context) error {
				if err := common.InitDBEngine(ctx); err != nil {
					return err
				}
				log.Info("PostgreSQL ready")
				return nil
			},
			Rollback: db.CloseEngine,
		},
		startup.Step{
			Name:     "Redis",
			Init:     common.InitRedis,
			Rollback: conn.CloseRedis,
		},
		startup.Step{
			Name: "access token verifier",
			Init: func(context.Context) error { return auth.Init() },
		},
	)
}

// NormalRoutes registers the versioned API tree, including Organization/Team.
func NormalRoutes() *web.Router {
	r := web.NewRouter()
	r.Mount("/api/v1", apiv1.Routes())
	return r
}

// PrepareWebRoutes compiles the HTTP route tree before opening DB/Redis.
// Panics during route registration are programmer errors, not runtime outages.
func PrepareWebRoutes() (http.Handler, error) {
	routes := NormalRoutes()
	if setting.API.EnableSwagger {
		swagger := api.SwaggerHandler(setting.Common.RunMode)
		if swagger == nil {
			return nil, fmt.Errorf("Swagger enabled in unsupported run mode %q", setting.Common.RunMode)
		}
		routes.Get("/swagger/*", swagger.ServeHTTP)
	}
	if err := routes.SetTrustedProxies(setting.Server.TrustedProxies); err != nil {
		return nil, fmt.Errorf("configure HTTP trusted proxies: %w", err)
	}
	return routes.WrapIngress(), nil
}

// NormalHandler attaches runtime admission to the prepared HTTP tree.
// Redis remains owned by modules/conn; the middleware only borrows its client.
func NormalHandler(routes http.Handler) (http.Handler, error) {
	if routes == nil {
		return nil, errors.New("HTTP routes must be prepared before resource startup")
	}
	client, err := conn.ProcessRedis()
	if err != nil {
		return nil, fmt.Errorf("IP admission Redis unavailable: %w", err)
	}
	limiter, err := ipactive.NewRedis(client, setting.IPActive.Policies)
	if err != nil {
		return nil, fmt.Errorf("configure IP admission: %w", err)
	}
	admission := ipactive.NewMiddleware(limiter)
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		req = req.WithContext(ipactive.WithMiddleware(req.Context(), admission))
		routes.ServeHTTP(w, req)
	}), nil
}
