package apiv1

import (
	"net/http"
	"time"

	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/modules/token"
	"github.com/datvtph41107/bdspro/routers/api/middleware"
	"github.com/datvtph41107/bdspro/routers/api/v1/org"
	"github.com/datvtph41107/bdspro/routers/api/v1/user"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

// Dependencies are constructed once by the HTTP server; route ownership is in this package.
type Dependencies struct {
	Auth         *user.AuthHandler
	Organization *org.Handler
	Google       *user.GoogleHandler
	Phone        *user.PhoneHandler
	TokenMaker   token.Maker
	RateRedis    *redis.Client
	Store        db.Store
}

// Routes declares the existing Gin HTTP surface using Gitea-style file ownership.
// Each registered method remains visible without a routing DSL or registry.
func Routes(root *gin.Engine, d Dependencies) {
	root.GET("/health", Health)
	registerPublicRoutes(root.Group(""), d)
}

func registerPublicRoutes(route *gin.RouterGroup, d Dependencies) {
	route.POST("/users",
		middleware.IPActiveSource(d.RateRedis, "register", 10, time.Hour),
		d.Auth.Register,
	)
	route.POST("/users/login",
		middleware.LoginAudit(),
		middleware.IPActiveSource(d.RateRedis, "login", 30, time.Minute),
		d.Auth.Login,
	)
	route.GET("/users/verify_email", d.Auth.VerifyEmailPage)
	route.POST("/users/verify_email",
		middleware.IPActiveSource(d.RateRedis, "verify_email", 60, time.Minute),
		d.Auth.VerifyEmail,
	)
	route.POST("/users/verify_email/resend",
		middleware.IPActiveSource(d.RateRedis, "verify_email_resend", 10, time.Hour),
		d.Auth.ResendVerificationEmail,
	)
	route.POST("/tokens/renew_access",
		middleware.IPActiveSource(d.RateRedis, "refresh", 60, time.Minute),
		d.Auth.Renew,
	)
	route.POST("/tokens/revoke",
		middleware.IPActiveSource(d.RateRedis, "revoke", 60, time.Minute),
		d.Auth.Revoke,
	)
	route.POST("/users/password/reset/request",
		middleware.IPActiveSource(d.RateRedis, "password_reset", 10, time.Hour),
		d.Auth.RequestPasswordReset,
	)
	route.GET("/users/password/reset", d.Auth.PasswordResetPage)
	route.POST("/users/password/reset",
		middleware.IPActiveSource(d.RateRedis, "password_reset_confirm", 20, time.Hour),
		d.Auth.ResetPassword,
	)

	if d.Google != nil {
		route.GET("/users/google/login/start",
			middleware.IPActiveSource(d.RateRedis, "google_login_start", 20, time.Minute),
			d.Google.StartLogin,
		)
		route.GET("/users/google/callback", d.Google.Callback)
	}
	if d.Phone != nil {
		route.POST("/users/phone/register/start",
			middleware.IPActiveSource(d.RateRedis, "phone_register_start", 10, time.Hour),
			d.Phone.StartRegister,
		)
		route.POST("/users/phone/register/verify",
			middleware.IPActiveSource(d.RateRedis, "phone_register_verify", 40, 10*time.Minute),
			d.Phone.VerifyRegister,
		)
		route.POST("/users/phone/login/start",
			middleware.IPActiveSource(d.RateRedis, "phone_login_start", 30, time.Minute),
			d.Phone.StartLogin,
		)
		route.POST("/users/phone/login/verify",
			middleware.IPActiveSource(d.RateRedis, "phone_login_verify", 40, 10*time.Minute),
			d.Phone.VerifyLogin,
		)
	}

	// Authentication is applied once to this group, before every child group.
	registerAuthenticatedRoutes(route.Group("", middleware.Authenticate(d.TokenMaker)), d)
}

func registerAuthenticatedRoutes(route *gin.RouterGroup, d Dependencies) {
	route.GET("/users/me", d.Auth.Me)
	route.GET("/users/me/sessions", d.Auth.ListSessions)
	route.DELETE("/users/me/sessions/:id", d.Auth.RevokeOwnSession)
	route.POST("/users/me/sessions/revoke_all", d.Auth.RevokeAllSessions)
	route.POST("/users/me/email/start",
		middleware.IPActiveSource(d.RateRedis, "email_link_start", 10, time.Hour),
		d.Auth.StartEmailLink,
	)
	route.POST("/users/me/email/verify",
		middleware.IPActiveSource(d.RateRedis, "email_link_verify", 20, time.Hour),
		d.Auth.ConfirmEmailLink,
	)

	registerOrganizationRoutes(route.Group("/organizations", middleware.VerifiedContact(d.Store)), d)

	if d.Google != nil {
		route.POST("/users/me/google/start",
			middleware.IPActiveSource(d.RateRedis, "google_link_start", 10, time.Minute),
			d.Google.StartLink,
		)
	}
	if d.Google != nil && d.Phone != nil {
		route.POST("/users/me/phone/google/start",
			middleware.VerifiedEmail(d.Store),
			middleware.IPActiveSource(d.RateRedis, "phone_google_bind_start", 10, time.Hour),
			d.Google.StartPhoneBind,
		)
	}
	if d.Phone != nil {
		registerPhoneBindRoutes(route.Group("/users/me/phone", middleware.VerifiedEmail(d.Store)), d)
	}
}

func registerOrganizationRoutes(route *gin.RouterGroup, d Dependencies) {
	route.GET("", d.Organization.List)
	route.GET("/:id", d.Organization.Get)
	route.POST("", d.Organization.Create)
	route.PATCH("/:id", d.Organization.Edit)
	route.GET("/:id/members", d.Organization.Members)
	route.POST("/:id/members", d.Organization.AddMember)
}

func registerPhoneBindRoutes(route *gin.RouterGroup, d Dependencies) {
	route.POST("/start",
		middleware.IPActiveSource(d.RateRedis, "phone_bind_start", 10, time.Hour),
		d.Phone.StartBind,
	)
	route.POST("/verify",
		middleware.IPActiveSource(d.RateRedis, "phone_bind_verify", 40, 10*time.Minute),
		d.Phone.VerifyBind,
	)
}

// @Summary Check API health
// @Tags System
// @Success 200 "Service is healthy"
// @Router /health [get]
func Health(c *gin.Context) {
	c.Status(http.StatusOK)
}
