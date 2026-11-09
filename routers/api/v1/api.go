package apiv1

import (
	"net/http"

	db "github.com/datvtph41107/bdspro/models/db"
	organization_model "github.com/datvtph41107/bdspro/models/organization"
	authcrypto "github.com/datvtph41107/bdspro/modules/auth"
	"github.com/datvtph41107/bdspro/modules/ipactive"
	api "github.com/datvtph41107/bdspro/modules/structs"
	"github.com/datvtph41107/bdspro/modules/web"
	"github.com/datvtph41107/bdspro/routers/api/v1/oauth"
	"github.com/datvtph41107/bdspro/routers/api/v1/org"
	"github.com/datvtph41107/bdspro/routers/api/v1/user"
	weboauth "github.com/datvtph41107/bdspro/routers/web/oauth"
	appctx "github.com/datvtph41107/bdspro/services/context"
	security "github.com/datvtph41107/bdspro/services/security"
	"github.com/gin-gonic/gin"
)

// Dependencies are constructed once by the HTTP server; route ownership is in this package.
type Dependencies struct {
	Auth          *user.AuthHandler
	Organization  *org.Handler
	OAuth2        *user.OAuth2Handler
	Authorization *oauth.Handler
	Browser       *weboauth.Handler
	PhoneEnabled  bool
	TokenMaker    authcrypto.Maker
	IPActive      *ipactive.Middleware
	Store         db.Store
	InviteKey     string
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
		security.LogLoginResponse(),
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

	if d.Authorization != nil {
		// One canonical method/route declaration. When browser OAuth is not
		// configured, the existing Bearer-JWT-only contract remains unchanged.
		preview := d.Authorization.AuthorizePreview
		approve := d.Authorization.Authorize
		authenticate := d.Authorization.Authenticate(d.TokenMaker)
		if d.Browser != nil {
			preview = d.Browser.AuthorizePreview
			approve = d.Browser.Authorize
			authenticate = d.Browser.Authenticate
			route.GET("/oauth/login", d.Browser.LoginPage)
			loginHandlers := []gin.HandlerFunc{d.Browser.Login}
			if d.IPActive != nil {
				loginHandlers = append([]gin.HandlerFunc{d.IPActive.For(ipactive.Login)}, loginHandlers...)
			}
			route.POST("/oauth/login", loginHandlers...)
			route.POST("/oauth/logout", d.Browser.RequireBrowserSession, d.Browser.Logout)
		}
		route.GET("/oauth/authorize", authenticate, preview)
		route.POST("/oauth/authorize", authenticate, approve)
		// OAuth2 authorization-server endpoints are distinct from OIDC client login.
		route.POST("/oauth/token",
			d.IPActive.For(ipactive.RenewAccess), d.Authorization.Token)
		route.POST("/oauth/revoke",
			d.IPActive.For(ipactive.RevokeToken), d.Authorization.Revoke)
		route.GET("/oauth/userinfo", d.Authorization.UserInfo)
	}
	if d.OAuth2 != nil {
		if d.Browser != nil && d.Browser.OIDCEnabled {
			route.GET("/oauth/login/oidc/start",
				d.IPActive.For(ipactive.StartOAuth2Login), d.OAuth2.StartBrowserLogin)
		}
		route.GET("/users/oauth2/login/start",
			d.IPActive.For(ipactive.StartOAuth2Login),
			d.OAuth2.StartLogin,
		)
		route.GET("/users/oauth2/callback", d.OAuth2.Callback)
	}
	if d.PhoneEnabled {
		route.POST("/users/phone/register/start",
			d.IPActive.For(ipactive.StartPhoneRegister),
			d.Auth.StartPhoneRegister,
		)
		route.POST("/users/phone/register/verify",
			d.IPActive.For(ipactive.VerifyPhoneRegister),
			d.Auth.VerifyPhoneRegister,
		)
		route.POST("/users/phone/login/start",
			d.IPActive.For(ipactive.StartPhoneLogin),
			d.Auth.StartPhoneLogin,
		)
		route.POST("/users/phone/login/verify",
			d.IPActive.For(ipactive.VerifyPhoneLogin),
			d.Auth.VerifyPhoneLogin,
		)
	}

	// Check the signed access principal and its live PostgreSQL session once
	// at ingress. Domain ownership and in-transaction authorization stay in
	// models/services; no parallel middleware executor is introduced.
	registerAuthenticatedRoutes(route.Group("",
		appctx.Authenticate(d.TokenMaker),
		appctx.RequireSession(d.Store),
	), d)
}

func registerAuthenticatedRoutes(route *gin.RouterGroup, d Dependencies) {
	profile := user.NewProfileHandler(d.Store)
	if d.Authorization != nil {
		route.GET("/oauth/clients", d.Authorization.ListClients)
		route.POST("/oauth/clients",
			appctx.VerifiedContact(d.Store), d.Authorization.CreateClient)
		route.DELETE("/oauth/clients/:id", d.Authorization.DeleteClient)
		// Existing user authorizations, not application ownership.
		route.GET("/users/me/oauth/authorizations", d.Authorization.ListAuthorizations)
		route.DELETE("/users/me/oauth/authorizations/:client_id", d.Authorization.RevokeAuthorization)
	}
	route.GET("/users/me", appctx.Wrap(profile.Show))
	route.PATCH("/users/me/profile", appctx.Wrap(profile.Update))
	route.GET("/users/me/settings", appctx.Wrap(profile.Settings))
	route.PATCH("/users/me/settings", appctx.Wrap(profile.UpdateSettings))
	// Recent successful session issuance evidence, scoped to the authenticated Account.
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

	// One team handler owns the immutable store/signing dependencies for both
	// organization-scoped and team-scoped routes; no request-time assembly.
	teams := org.NewTeamHandler(d.Store, d.InviteKey)
	registerOrgRoutes(route.Group("/orgs", appctx.VerifiedContact(d.Store)), d, teams)
	registerTeamRoutes(route.Group("/teams", appctx.VerifiedContact(d.Store)), d, teams)

	if d.OAuth2 != nil {
		route.POST("/users/me/oauth2/start",
			d.IPActive.For(ipactive.StartOAuth2Link),
			d.OAuth2.StartLink,
		)
	}
	if d.OAuth2 != nil && d.PhoneEnabled {
		route.POST("/users/me/phone/oauth2/start",
			appctx.VerifiedEmail(d.Store),
			d.IPActive.For(ipactive.StartOAuth2PhoneBind),
			d.OAuth2.StartPhoneBind,
		)
	}
	if d.PhoneEnabled {
		registerPhoneBindRoutes(route.Group("/users/me/phone", appctx.VerifiedEmail(d.Store)), d)
	}
}

func registerOrgRoutes(route *gin.RouterGroup, d Dependencies, teams *org.TeamHandler) {
	route.GET("", appctx.Wrap(org.List))
	route.GET("/:id/members", appctx.Wrap(d.Organization.Members))
	route.POST("/:id/members", appctx.Wrap(d.Organization.AddMember))
	route.DELETE("/:id/members/:person_id", appctx.Wrap(d.Organization.RemoveMember))
	route.PUT("/:id/owners/:person_id", appctx.Wrap(d.Organization.PromoteOwner))
	route.DELETE("/:id/owners/:person_id", appctx.Wrap(d.Organization.DemoteOwner))
	route.POST("/:id/owners/transfer", appctx.Wrap(d.Organization.TransferOwner))
	m := web.NewRouter[*appctx.APIContext](route, appctx.GetAPIContext)
	m.Post("", reqToken(), bind(api.CreateOrgOption{}), org.Create)
	m.Group("/:id", func() {
		m.Get("", reqToken(), org.Get)
		m.Get("/teams", reqToken(), reqOrgMembership(), org.ListTeams)
	}, orgAssignment(nil))
	m.Group("/:id", func() {
		m.Patch("", reqToken(), reqOrgOwnership(organization_model.ErrOrganizationOwnerRequired), bind(api.EditOrgOption{}), org.Edit)
	}, orgAssignment(organization_model.ErrOrganizationOwnerRequired))
	m.Group("/:id", func() {
		m.Group("/teams", func() {
			m.Post("", reqToken(), reqOrgMembership(), reqOrgOwnership(), bind(api.CreateTeamOption{}), org.CreateTeam)
		})
	}, orgAssignment(organization_model.ErrTeamOwnerRequired))
}

func registerTeamRoutes(route *gin.RouterGroup, d Dependencies, teams *org.TeamHandler) {
	route.POST("/invites/accept",
		d.IPActive.For(ipactive.AcceptTeamInvite),
		appctx.Wrap(teams.AcceptInvite),
	)
	m := web.NewRouter[*appctx.APIContext](route, appctx.GetAPIContext)
	m.Group("/:team_id", func() {
		m.Get("", reqToken(), org.GetTeam)
	}, teamAssignment())
	route.PATCH("/:team_id", appctx.Wrap(teams.Edit))
	route.GET("/:team_id/members", appctx.Wrap(teams.Members))
	route.POST("/:team_id/members", appctx.Wrap(teams.AddMember))
	route.DELETE("/:team_id/members/:person_id", appctx.Wrap(teams.RemoveMember))
	route.POST("/:team_id/invites",
		d.IPActive.For(ipactive.CreateTeamInvite),
		appctx.Wrap(teams.CreateInvite),
	)
	route.DELETE("/:team_id/invites/:invite_id", appctx.Wrap(teams.RevokeInvite))
}

func registerPhoneBindRoutes(route *gin.RouterGroup, d Dependencies) {
	route.POST("/start",
		d.IPActive.For(ipactive.StartPhoneBind),
		d.Auth.StartPhoneBind,
	)
	route.POST("/verify",
		d.IPActive.For(ipactive.VerifyPhoneBind),
		d.Auth.VerifyPhoneBind,
	)
}

// @Summary Check API health
// @Tags System
// @Success 200 "Service is healthy"
// @Router /health [get]
func Health(c *gin.Context) {
	c.Status(http.StatusOK)
}
