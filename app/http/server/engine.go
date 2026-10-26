package server

import (
	"errors"
	"net/http"
	"net/url"
	"time"

	apicontroller "github.com/datvtph41107/bdspro/app/http/controllers/api"
	authcontroller "github.com/datvtph41107/bdspro/app/http/controllers/auth"
	"github.com/datvtph41107/bdspro/app/http/middleware"
	db "github.com/datvtph41107/bdspro/db/sqlc"
	"github.com/datvtph41107/bdspro/routes"
	"github.com/datvtph41107/bdspro/token"
	"github.com/datvtph41107/bdspro/util"
	"github.com/datvtph41107/bdspro/worker"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

// NewServer constructs all HTTP dependencies *once*, then mounts native Gin
// handlers. No route binding table or dynamic controller resolution is used.
func NewServer(config util.Config, store db.Store, tasks worker.TaskDistributor, redisClient *redis.Client) (*Server, error) {
	if redisClient == nil {
		return nil, errors.New("authentication rate limiter requires Redis")
	}
	maker, err := token.NewJWTMaker(config.TokenSymmetricKey)
	if err != nil {
		return nil, err
	}
	limiter := middleware.NewRateLimiter(redisClient)
	scheduler := worker.NewVerificationScheduler(tasks)
	auth := authcontroller.NewController(store, scheduler, maker, config, limiter)
	organization := apicontroller.NewOrganizationController(store)

	var google *authcontroller.GoogleController
	if config.GoogleClientID != "" {
		redirect, err := url.Parse(config.GoogleRedirectURI)
		if err != nil {
			return nil, err
		}
		google = authcontroller.NewGoogleController(store, maker, config, redisClient,
			&http.Client{Timeout: 10 * time.Second}, redirect.Scheme == "https")
	}
	var phone *authcontroller.PhoneController
	if config.Environment == "development" {
		phone = authcontroller.NewPhoneController(store, store, maker, config, limiter)
	}

	router := gin.New()
	if err := router.SetTrustedProxies(nil); err != nil {
		return nil, err
	}
	router.Use(
		middleware.RequestID(),
		middleware.RequestLog(),
		middleware.Recovery(),
		limiter.RequestMonitor(),
		middleware.RequestIngress(),
	)
	routes.RegisterAuth(router, auth, limiter)
    routes.RegisterAPI(router, auth, organization, maker)
    if google != nil {
        routes.RegisterGoogle(router, google, maker, limiter)
    }
    if phone != nil {
        routes.RegisterPhone(router, phone, maker, limiter)
    }
	return &Server{config: config, router: router}, nil
}
