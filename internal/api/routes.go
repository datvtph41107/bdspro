package api

import (
	"net/http"
	"time"

	"github.com/datvtph41107/bdspro/internal/api/middleware"
	"github.com/gin-gonic/gin"
)

// registerRoutes declares the HTTP endpoints on Gin directly.
// Handlers and middleware are created once when the server starts.
func (server *Server) registerRoutes(root *gin.Engine) {
	api := root.Group(apiPrefix)
	sourceLimit := func(operation string, attempts int64, window time.Duration) gin.HandlerFunc {
		return middleware.IPActiveSource(server.authRateRedis, operation, attempts, window)
	}

	root.GET("/health", Health)

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
	api.POST("/users/verify_email/resend",
		sourceLimit("verify_email_resend", 10, time.Hour),
		server.authHandler.ResendVerificationEmail,
	)
	api.GET("/users/verify_email", server.authHandler.VerifyEmailPage)
	api.POST("/users/verify_email",
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

	api.POST("/users/password/reset/request",
		sourceLimit("password_reset", 10, time.Hour),
		server.authHandler.RequestPasswordReset,
	)
	api.GET("/users/password/reset", server.authHandler.PasswordResetPage)
	api.POST("/users/password/reset",
		sourceLimit("password_reset_confirm", 20, time.Hour),
		server.authHandler.ResetPassword,
	)

	// Account authentication is inherited by every nested route group.
	authenticated := api.Group("", middleware.Authenticate(server.tokenMaker))
	{
		authenticated.GET("/users/me", server.authHandler.Me)
		authenticated.GET("/users/me/sessions", server.authHandler.ListSessions)
		authenticated.DELETE("/users/me/sessions/:id", server.authHandler.RevokeOwnSession)
		authenticated.POST("/users/me/sessions/revoke_all", server.authHandler.RevokeAllSessions)
		authenticated.POST("/users/me/email/start",
			sourceLimit("email_link_start", 10, time.Hour),
			server.authHandler.StartEmailLink,
		)
		authenticated.POST("/users/me/email/verify",
			sourceLimit("email_link_verify", 20, time.Hour),
			server.authHandler.ConfirmEmailLink,
		)

		organizations := authenticated.Group("/organizations", middleware.VerifiedContact(server.store))
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

	// OAuth confirmation proves control of the Google identity already linked
	// to this account before the phone verification challenge can be issued.
	if server.googleHandler != nil && server.phoneHandler != nil {
		authenticated.POST("/users/me/phone/google/start",
			middleware.VerifiedEmail(server.store),
			sourceLimit("phone_google_bind_start", 10, time.Hour),
			server.googleHandler.StartPhoneBind,
		)
	}

	// Phone verification is enabled only when a safe delivery channel is configured.
	if server.phoneHandler != nil {
		api.POST("/users/phone/register/start",
			sourceLimit("phone_register_start", 10, time.Hour),
			server.phoneHandler.StartRegister,
		)
		api.POST("/users/phone/register/verify",
			sourceLimit("phone_register_verify", 40, 10*time.Minute),
			server.phoneHandler.VerifyRegister,
		)
		api.POST("/users/phone/login/start",
			sourceLimit("phone_login_start", 30, time.Minute),
			server.phoneHandler.StartLogin,
		)
		api.POST("/users/phone/login/verify",
			sourceLimit("phone_login_verify", 40, 10*time.Minute),
			server.phoneHandler.VerifyLogin,
		)
		phone := authenticated.Group("/users/me/phone", middleware.VerifiedEmail(server.store))
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

// @Summary Check API health
// @Tags System
// @Success 200 "Service is healthy"
// @Router /health [get]
func Health(c *gin.Context) {
	c.Status(http.StatusOK)
}
