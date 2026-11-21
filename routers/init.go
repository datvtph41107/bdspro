// Package routers composes the HTTP infrastructure without domain services.
package routers

import (
	"context"
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
func InitWebInstalled(ctx context.Context, cfg setting.WebConfig) error {
	return startup.Run(ctx,
		startup.Step{
			Name: "PostgreSQL",
			Init: func(ctx context.Context) error {
				if err := common.InitDBEngine(ctx, cfg.Common.RunMode, cfg.Database); err != nil {
					return err
				}
				log.Info("PostgreSQL ready")
				return nil
			},
			Rollback: db.CloseEngine,
		},
		startup.Step{
			Name: "Redis",
			Init: func(ctx context.Context) error {
				return common.InitRedis(ctx, cfg.Redis.Address)
			},
			Rollback: conn.CloseRedis,
		},
		startup.Step{
			Name: "access token verifier",
			Init: func(context.Context) error { return auth.Init(cfg.Security.TokenSymmetricKey) },
		},
	)
}

// NormalRoutes registers the versioned API tree, including Organization/Team.
func NormalRoutes() *web.Router {
	r := web.NewRouter()
	r.Mount("/api/v1", apiv1.Routes())
	return r
}

func NormalHandler() (http.Handler, error) {
	// Borrow the process-owned Redis client. Security admission shares its
	// lifecycle and fails startup if the configured limiter cannot be built.
	client, err := conn.ProcessRedis()
	if err != nil {
		return nil, fmt.Errorf("IP admission Redis unavailable: %w", err)
	}
	limiter, err := ipactive.NewRedis(client, setting.IPActive.Policies)
	if err != nil {
		return nil, fmt.Errorf("configure IP admission: %w", err)
	}
	admission := ipactive.NewMiddleware(limiter)
	routes := NormalRoutes()
	if setting.API.EnableSwagger {
		swagger := api.SwaggerHandler(setting.Common.RunMode)
		routes.Get("/swagger/*", swagger.ServeHTTP)
	}
	if err := routes.SetTrustedProxies(setting.Server.TrustedProxies); err != nil {
		return nil, fmt.Errorf("configure HTTP trusted proxies: %w", err)
	}
	handler := routes.WrapIngress()

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		req = req.WithContext(ipactive.WithMiddleware(req.Context(), admission))
		handler.ServeHTTP(w, req)
	}), nil
}
