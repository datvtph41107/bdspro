package api

import (
	"net/http"
	"time"

	"github.com/datvtph41107/bdspro/internal/api/middleware"
	"github.com/gin-gonic/gin"
)

// registerRoutes uses Gin's native groups and middleware inheritance.
// Each registered method remains visible without a routing DSL or registry.
func (s *Server) registerRoutes(root *gin.Engine) {
	root.GET("/health", Health)
	s.registerPublicRoutes(root.Group(apiPrefix))
}

func (s *Server) registerPublicRoutes(route *gin.RouterGroup) {
	route.POST("/users",
		middleware.IPActiveSource(s.authRateRedis, "register", 10, time.Hour),
		s.authHandler.Register,
	)
	route.POST("/users/login",
		middleware.LoginAudit(),
		middleware.IPActiveSource(s.authRateRedis, "login", 30, time.Minute),
		s.authHandler.Login,
	)
	route.GET("/users/verify_email", s.authHandler.VerifyEmailPage)
	route.POST("/users/verify_email",
		middleware.IPActiveSource(s.authRateRedis, "verify_email", 60, time.Minute),
		s.authHandler.VerifyEmail,
	)
	route.POST("/users/verify_email/resend",
		middleware.IPActiveSource(s.authRateRedis, "verify_email_resend", 10, time.Hour),
		s.authHandler.ResendVerificationEmail,
	)
	route.POST("/tokens/renew_access",
		middleware.IPActiveSource(s.authRateRedis, "refresh", 60, time.Minute),
		s.authHandler.Renew,
	)
	route.POST("/tokens/revoke",
		middleware.IPActiveSource(s.authRateRedis, "revoke", 60, time.Minute),
		s.authHandler.Revoke,
	)
	route.POST("/users/password/reset/request",
		middleware.IPActiveSource(s.authRateRedis, "password_reset", 10, time.Hour),
		s.authHandler.RequestPasswordReset,
	)
	route.GET("/users/password/reset", s.authHandler.PasswordResetPage)
	route.POST("/users/password/reset",
		middleware.IPActiveSource(s.authRateRedis, "password_reset_confirm", 20, time.Hour),
		s.authHandler.ResetPassword,
	)

	if s.googleHandler != nil {
		route.GET("/users/google/login/start",
			middleware.IPActiveSource(s.authRateRedis, "google_login_start", 20, time.Minute),
			s.googleHandler.StartLogin,
		)
		route.GET("/users/google/callback", s.googleHandler.Callback)
	}
	if s.phoneHandler != nil {
		route.POST("/users/phone/register/start",
			middleware.IPActiveSource(s.authRateRedis, "phone_register_start", 10, time.Hour),
			s.phoneHandler.StartRegister,
		)
		route.POST("/users/phone/register/verify",
			middleware.IPActiveSource(s.authRateRedis, "phone_register_verify", 40, 10*time.Minute),
			s.phoneHandler.VerifyRegister,
		)
		route.POST("/users/phone/login/start",
			middleware.IPActiveSource(s.authRateRedis, "phone_login_start", 30, time.Minute),
			s.phoneHandler.StartLogin,
		)
		route.POST("/users/phone/login/verify",
			middleware.IPActiveSource(s.authRateRedis, "phone_login_verify", 40, 10*time.Minute),
			s.phoneHandler.VerifyLogin,
		)
	}

	// Authentication is applied once to this group, before every child group.
	s.registerAuthenticatedRoutes(route.Group("", middleware.Authenticate(s.tokenMaker)))
}

func (s *Server) registerAuthenticatedRoutes(route *gin.RouterGroup) {
	route.GET("/users/me", s.authHandler.Me)
	route.GET("/users/me/sessions", s.authHandler.ListSessions)
	route.DELETE("/users/me/sessions/:id", s.authHandler.RevokeOwnSession)
	route.POST("/users/me/sessions/revoke_all", s.authHandler.RevokeAllSessions)
	route.POST("/users/me/email/start",
		middleware.IPActiveSource(s.authRateRedis, "email_link_start", 10, time.Hour),
		s.authHandler.StartEmailLink,
	)
	route.POST("/users/me/email/verify",
		middleware.IPActiveSource(s.authRateRedis, "email_link_verify", 20, time.Hour),
		s.authHandler.ConfirmEmailLink,
	)

	s.registerOrganizationRoutes(route.Group("/organizations", middleware.VerifiedContact(s.store)))

	if s.googleHandler != nil {
		route.POST("/users/me/google/start",
			middleware.IPActiveSource(s.authRateRedis, "google_link_start", 10, time.Minute),
			s.googleHandler.StartLink,
		)
	}
	if s.googleHandler != nil && s.phoneHandler != nil {
		route.POST("/users/me/phone/google/start",
			middleware.VerifiedEmail(s.store),
			middleware.IPActiveSource(s.authRateRedis, "phone_google_bind_start", 10, time.Hour),
			s.googleHandler.StartPhoneBind,
		)
	}
	if s.phoneHandler != nil {
		s.registerPhoneBindRoutes(route.Group("/users/me/phone", middleware.VerifiedEmail(s.store)))
	}
}

func (s *Server) registerOrganizationRoutes(route *gin.RouterGroup) {
	route.GET("", s.organizationHandler.Index)
	route.GET("/:id", s.organizationHandler.Show)
	route.POST("", s.organizationHandler.Store)
	route.PATCH("/:id", s.organizationHandler.Update)
	route.GET("/:id/members", s.organizationHandler.Members)
	route.POST("/:id/members", s.organizationHandler.AddMember)
}

func (s *Server) registerPhoneBindRoutes(route *gin.RouterGroup) {
	route.POST("/start",
		middleware.IPActiveSource(s.authRateRedis, "phone_bind_start", 10, time.Hour),
		s.phoneHandler.StartBind,
	)
	route.POST("/verify",
		middleware.IPActiveSource(s.authRateRedis, "phone_bind_verify", 40, 10*time.Minute),
		s.phoneHandler.VerifyBind,
	)
}

// @Summary Check API health
// @Tags System
// @Success 200 "Service is healthy"
// @Router /health [get]
func Health(c *gin.Context) {
	c.Status(http.StatusOK)
}
