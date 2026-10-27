package routes

import (
	"net/http"
	"time"

	"github.com/datvtph41107/bdspro/internal/api/handler"
	"github.com/gin-gonic/gin"
)

// Handlers are constructed at startup. No reflection or container resolution.
type Handlers struct {
	Auth         *handler.AuthHandler
	Organization *handler.OrganizationHandler
	Google       *handler.GoogleHandler
	Phone        *handler.PhoneHandler
}

// Policies are concrete Gin middleware and startup-only middleware factories.
type Policies struct {
	Authenticate gin.HandlerFunc
	LoginAudit   gin.HandlerFunc
	SourceLimit  func(string, int64, time.Duration) gin.HandlerFunc
}

// API owns the entire HTTP route contract. Base prefix is supplied by the
// application router, like Laravel RouteServiceProvider's route group.
// All paths below preserve the existing unversioned public contract.
func API(root *gin.Engine, api *gin.RouterGroup, h Handlers, mw Policies) {
	// System endpoint (outside any future API version prefix).
	root.GET("/health", func(c *gin.Context) { c.Status(http.StatusOK) })

	// Public account and session endpoints.
	api.POST("/users",
		mw.SourceLimit("register", 10, time.Hour),
		h.Auth.Register,
	)
	api.POST("/users/login",
		mw.LoginAudit,
		mw.SourceLimit("login", 30, time.Minute),
		h.Auth.Login,
	)
	api.GET("/users/verify_email",
		mw.SourceLimit("verify_email", 60, time.Minute),
		h.Auth.VerifyEmail,
	)
	api.POST("/tokens/renew_access",
		mw.SourceLimit("refresh", 60, time.Minute),
		h.Auth.Renew,
	)
	api.POST("/tokens/revoke",
		mw.SourceLimit("revoke", 60, time.Minute),
		h.Auth.Revoke,
	)

	// Account authentication is inherited by every nested route group.
	authenticated := api.Group("", mw.Authenticate)
	{
		authenticated.GET("/users/me", h.Auth.Me)

		organizations := authenticated.Group("/organizations")
		{
			organizations.POST("", h.Organization.Create)
			organizations.GET("", h.Organization.Index)
			organizations.POST("/:id/members", h.Organization.AddMember)
			organizations.PATCH("/:id", h.Organization.Update)
		}
	}

	// Google OAuth exists only when the configured client is available.
	if h.Google != nil {
		authenticated.POST("/users/me/google/start",
			mw.SourceLimit("google_link_start", 10, time.Minute),
			h.Google.StartLink,
		)
		api.GET("/users/google/login/start",
			mw.SourceLimit("google_login_start", 20, time.Minute),
			h.Google.StartLogin,
		)
		api.GET("/users/google/callback", h.Google.Callback)
	}

	// Phone OTP is explicitly limited to development in application startup.
	if h.Phone != nil {
		api.POST("/users/phone/login/start",
			mw.SourceLimit("phone_login_start", 10, time.Hour),
			h.Phone.StartLogin,
		)
		api.POST("/users/phone/login/verify",
			mw.LoginAudit,
			mw.SourceLimit("phone_login_verify", 40, 10*time.Minute),
			h.Phone.VerifyLogin,
		)

		phone := authenticated.Group("/users/me/phone")
		{
			phone.POST("/start",
				mw.SourceLimit("phone_bind_start", 10, time.Hour),
				h.Phone.StartBind,
			)
			phone.POST("/verify",
				mw.SourceLimit("phone_bind_verify", 40, 10*time.Minute),
				h.Phone.VerifyBind,
			)
		}
	}
}
