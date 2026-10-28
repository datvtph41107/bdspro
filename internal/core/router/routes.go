package router

import (
	"net/http"
	"time"

	"github.com/datvtph41107/bdspro/internal/core/middleware"
	"github.com/gin-gonic/gin"
)

// registerRoutes declares the HTTP endpoints on Gin directly.
// Handlers and middleware are created once when the server starts.
func (server *Server) registerRoutes(root *gin.Engine) {
	api := root.Group(apiPrefix)
	sourceLimit := func(operation string, attempts int64, window time.Duration) gin.HandlerFunc {
		return middleware.IPActiveSource(server.authRateRedis, operation, attempts, window)
	}
	// System endpoint (outside any future API version prefix).
	root.GET("/health", func(c *gin.Context) { c.Status(http.StatusOK) })

	// Public account and session endpoints.
	api.POST("/users",
		sourceLimit("register", 10, time.Hour),
		server.authHandler.Register,
	)
	api.POST("/users/login",
		middleware.LoginAudit(),
		sourceLimit("login", 30, time.Minute),
		server.authHandler.Login,
	)
	api.GET("/users/verify_email",
		sourceLimit("verify_email", 60, time.Minute),
		server.authHandler.VerifyEmail,
	)
	api.POST("/tokens/renew_access",
		sourceLimit("refresh", 60, time.Minute),
		server.authHandler.Renew,
	)
	api.POST("/tokens/revoke",
		sourceLimit("revoke", 60, time.Minute),
		server.authHandler.Revoke,
	)

	// Account authentication is inherited by every nested route group.
	authenticated := api.Group("", middleware.Authenticate(server.tokenMaker))
	{
		authenticated.GET("/users/me", server.authHandler.Me)

		organizations := authenticated.Group("/organizations")
		{
			organizations.POST("", server.organizationHandler.Create)
			organizations.GET("", server.organizationHandler.Index)
			organizations.POST("/:id/members", server.organizationHandler.AddMember)
			organizations.PATCH("/:id", server.organizationHandler.Update)
		}
	}

	// Google OAuth exists only when the configured client is available.
	if server.googleHandler != nil {
		authenticated.POST("/users/me/google/start",
			sourceLimit("google_link_start", 10, time.Minute),
			server.googleHandler.StartLink,
		)
		api.GET("/users/google/login/start",
			sourceLimit("google_login_start", 20, time.Minute),
			server.googleHandler.StartLogin,
		)
		api.GET("/users/google/callback", server.googleHandler.Callback)
	}

	// Phone OTP is explicitly limited to development in application startup.
	if server.phoneHandler != nil {
		api.POST("/users/phone/login/start",
			sourceLimit("phone_login_start", 10, time.Hour),
			server.phoneHandler.StartLogin,
		)
		api.POST("/users/phone/login/verify",
			middleware.LoginAudit(),
			sourceLimit("phone_login_verify", 40, 10*time.Minute),
			server.phoneHandler.VerifyLogin,
		)

		phone := authenticated.Group("/users/me/phone")
		{
			phone.POST("/start",
				sourceLimit("phone_bind_start", 10, time.Hour),
				server.phoneHandler.StartBind,
			)
			phone.POST("/verify",
				sourceLimit("phone_bind_verify", 40, 10*time.Minute),
				server.phoneHandler.VerifyBind,
			)
		}
	}
}
