// Package routers builds the HTTP application using Gin and existing modules.
package routers

import (
	"fmt"
	"os"
	"time"

	"github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/modules/log"
	"github.com/datvtph41107/bdspro/modules/setting"
	"github.com/datvtph41107/bdspro/routers/api"
	"github.com/datvtph41107/bdspro/routers/api/middleware"
	"github.com/datvtph41107/bdspro/routers/api/response"
	apiv1 "github.com/datvtph41107/bdspro/routers/api/v1"
	"github.com/datvtph41107/bdspro/routers/common"
	"github.com/datvtph41107/bdspro/services/mail"
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

// NormalRoutes installs the shared ingress chain and mounts the existing API
// and local Swagger UI. Gin remains the only routing engine; no new wrappers.
func NormalRoutes(server *Server) *gin.Engine {
	router := gin.New()
	// Safe default for internal/registration-only routers: never trust forwarded
	// headers. NewServer configures validated deployment peers before serving.
	_ = router.SetTrustedProxies(nil)

	router.Use(common.RequestID())
	router.Use(accessLogger(), gin.CustomRecoveryWithWriter(nil, func(ctx *gin.Context, recovered any) {
		path := ctx.FullPath()
		if path == "" {
			path = "<unmatched>"
		}
		log.Ctx(ctx.Request.Context()).Error().
			Str("method", ctx.Request.Method).
			Str("path", path).
			Str("panic_type", fmt.Sprintf("%T", recovered)).
			Msg("http.panic")
		response.InternalError(ctx)
	}))

	router.Use(
		middleware.RequestMonitor(server.authRateRedis),
		middleware.RequestIngress(),
	)

	apiv1.Routes(router, apiv1.Dependencies{
		Auth:          server.authHandler,
		Organization:  server.organizationHandler,
		OAuth2:        server.oauth2Handler,
		Authorization: server.authorizationHandler,
		Browser: server.browserHandler,
		PhoneEnabled:  server.phoneEnabled,
		TokenMaker:    server.tokenMaker,
		IPActive:      server.ipActive,
		Store:         server.store,
		InviteKey:     server.config.TokenSymmetricKey,
	})
	api.RegisterSwagger(router, server.config.Environment)

	return router
}

// accessLogger writes safe route-template outcomes without raw identifiers,
// token-bearing query strings or duplicate HTTP diagnostics.
// accessLogger records one safe request outcome, distinct from a failure
// diagnostic. Route templates avoid token-bearing query strings and raw IDs.
func accessLogger() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		startedAt := time.Now()
		ctx.Next()

		path := ctx.FullPath()
		if path == "" {
			path = "<unmatched>"
		}
		event := log.Ctx(ctx.Request.Context()).Info()
		if code := ctx.GetString(response.ErrorCodeKey); code != "" {
			event = event.Str("error_code", code)
		}
		event.Str("method", ctx.Request.Method).
			Str("path", path).
			Int("status", ctx.Writer.Status()).
			Dur("duration", time.Since(startedAt)).
			Msg("http.request")
	}
}
