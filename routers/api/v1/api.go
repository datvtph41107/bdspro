package apiv1

import (
	"net/http"

	db "github.com/datvtph41107/bdspro/models/db"
	authcrypto "github.com/datvtph41107/bdspro/modules/auth"
	"github.com/datvtph41107/bdspro/modules/ipactive"
	"github.com/datvtph41107/bdspro/routers/api/middleware"
	"github.com/datvtph41107/bdspro/routers/api/v1/org"
	"github.com/datvtph41107/bdspro/routers/api/v1/user"
	"github.com/gin-gonic/gin"
)

// Dependencies are constructed once by the HTTP server; route ownership is in this package.
type Dependencies struct {
	Auth         *user.AuthHandler
	Organization *org.Handler
	OAuth2       *user.OAuth2Handler
	Phone        *user.PhoneHandler
	TokenMaker   authcrypto.Maker
	IPActive     *ipactive.Middleware
	Store        db.Store
	InviteKey    string
}

// Routes declares the existing Gin HTTP surface using Gitea-style file ownership.
// Each registered method remains visible without a routing DSL or registry.
func Routes(root *gin.Engine, d Dependencies) {
	root.GET("/health", Health)
	registerPublicRoutes(root.Group(""), d)
}

func registerPublicRoutes(route *gin.RouterGroup, d Dependencies) {
	route.POST("/users",
		d.IPActive.For(ipactive.Register),
		d.Auth.Register,
	)
	route.POST("/users/login",
		middleware.LoginAudit(),
		d.IPActive.For(ipactive.Login),
		d.Auth.Login,
	)
	route.GET("/users/verify_email", d.Auth.VerifyEmailPage)
	route.POST("/users/verify_email",
		d.IPActive.For(ipactive.VerifyEmail),
		d.Auth.VerifyEmail,
	)
	route.POST("/users/verify_email/resend",
		d.IPActive.For(ipactive.ResendVerificationEmail),
		d.Auth.ResendVerificationEmail,
	)
	route.POST("/tokens/renew_access",
		d.IPActive.For(ipactive.RenewAccess),
		d.Auth.Renew,
	)
	route.POST("/tokens/revoke",
		d.IPActive.For(ipactive.RevokeToken),
		d.Auth.Revoke,
	)
	route.POST("/users/password/reset/request",
		d.IPActive.For(ipactive.RequestPasswordReset),
		d.Auth.RequestPasswordReset,
	)
	route.GET("/users/password/reset", d.Auth.PasswordResetPage)
	route.POST("/users/password/reset",
		d.IPActive.For(ipactive.ConfirmPasswordReset),
		d.Auth.ResetPassword,
	)

	if d.OAuth2 != nil {
		route.GET("/users/oauth2/login/start",
			d.IPActive.For(ipactive.StartOAuth2Login),
			d.OAuth2.StartLogin,
		)
		route.GET("/users/oauth2/callback", d.OAuth2.Callback)
	}
	if d.Phone != nil {
		route.POST("/users/phone/register/start",
			d.IPActive.For(ipactive.StartPhoneRegister),
			d.Phone.StartRegister,
		)
		route.POST("/users/phone/register/verify",
			d.IPActive.For(ipactive.VerifyPhoneRegister),
			d.Phone.VerifyRegister,
		)
		route.POST("/users/phone/login/start",
			d.IPActive.For(ipactive.StartPhoneLogin),
			d.Phone.StartLogin,
		)
		route.POST("/users/phone/login/verify",
			d.IPActive.For(ipactive.VerifyPhoneLogin),
			d.Phone.VerifyLogin,
		)
	}

	// Authentication is applied once to this group, before every child group.
	registerAuthenticatedRoutes(route.Group("", middleware.Authenticate(d.TokenMaker)), d)
}

func registerAuthenticatedRoutes(route *gin.RouterGroup, d Dependencies) {
	profile := user.NewProfileHandler(d.Store)
	route.GET("/users/me", profile.Show)
	route.PATCH("/users/me/profile", profile.Update)
	route.GET("/users/me/settings", profile.Settings)
	route.PATCH("/users/me/settings", profile.UpdateSettings)
	route.GET("/users/me/connections", d.Auth.GetConnections)
	route.GET("/users/me/sessions", d.Auth.ListSessions)
	route.DELETE("/users/me/sessions/:id", d.Auth.RevokeOwnSession)
	route.POST("/users/me/sessions/revoke_all", d.Auth.RevokeAllSessions)
	route.POST("/users/me/email/start",
		d.IPActive.For(ipactive.StartEmailLink),
		d.Auth.StartEmailLink,
	)
	route.POST("/users/me/email/verify",
		d.IPActive.For(ipactive.ConfirmEmailLink),
		d.Auth.ConfirmEmailLink,
	)

	registerOrganizationRoutes(route.Group("/organizations", middleware.VerifiedContact(d.Store)), d)
	registerTeamRoutes(route.Group("/teams", middleware.VerifiedContact(d.Store)), d)

	if d.OAuth2 != nil {
		route.POST("/users/me/oauth2/start",
			d.IPActive.For(ipactive.StartOAuth2Link),
			d.OAuth2.StartLink,
		)
	}
	if d.OAuth2 != nil && d.Phone != nil {
		route.POST("/users/me/phone/oauth2/start",
			middleware.VerifiedEmail(d.Store),
			d.IPActive.For(ipactive.StartOAuth2PhoneBind),
			d.OAuth2.StartPhoneBind,
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
	teams := org.NewTeamHandler(d.Store, d.InviteKey)
	route.GET("/:id/teams", teams.List)
	route.POST("/:id/teams", teams.Create)
}

func registerTeamRoutes(route *gin.RouterGroup, d Dependencies) {
	teams := org.NewTeamHandler(d.Store, d.InviteKey)
	route.POST("/invites/accept",
		d.IPActive.For(ipactive.AcceptTeamInvite),
		teams.AcceptInvite,
	)
	route.GET("/:team_id", teams.Get)
	route.PATCH("/:team_id", teams.Edit)
	route.GET("/:team_id/members", teams.Members)
	route.POST("/:team_id/members", teams.AddMember)
	route.DELETE("/:team_id/members/:person_id", teams.RemoveMember)
	route.POST("/:team_id/invites",
		d.IPActive.For(ipactive.CreateTeamInvite),
		teams.CreateInvite,
	)
	route.DELETE("/:team_id/invites/:invite_id", teams.RevokeInvite)
}

func registerPhoneBindRoutes(route *gin.RouterGroup, d Dependencies) {
	route.POST("/start",
		d.IPActive.For(ipactive.StartPhoneBind),
		d.Phone.StartBind,
	)
	route.POST("/verify",
		d.IPActive.For(ipactive.VerifyPhoneBind),
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
