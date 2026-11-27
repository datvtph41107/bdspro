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
	"github.com/datvtph41107/bdspro/services/phonedelivery"
)

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

func NormalRoutes() *web.Router {
	r := web.NewRouter()
	r.Mount("/api/v1", apiv1.Routes())
	return r
}

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
	return requestLogging(routes.WrapIngress()), nil
}

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
	phoneDelivery, err := phonedelivery.New(setting.PhoneDelivery, setting.SMS, setting.Common.RunMode)
	if err != nil {
		return nil, fmt.Errorf("configure phone delivery: %w", err)
	}
	admission := ipactive.NewMiddleware(limiter)
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		req = req.WithContext(ipactive.WithMiddleware(req.Context(), admission))
		req = req.WithContext(phonedelivery.WithService(req.Context(), phoneDelivery))
		routes.ServeHTTP(w, req)
	}), nil
}
