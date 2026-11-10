package apiv1

import (
	"errors"
	"net/http"

	db "github.com/datvtph41107/bdspro/models/db"
	organization_model "github.com/datvtph41107/bdspro/models/organization"
	authcrypto "github.com/datvtph41107/bdspro/modules/auth"
	"github.com/datvtph41107/bdspro/modules/ipactive"
	api "github.com/datvtph41107/bdspro/modules/structs"
	"github.com/datvtph41107/bdspro/modules/web"
	"github.com/datvtph41107/bdspro/modules/web/middleware"
	"github.com/datvtph41107/bdspro/routers/api/v1/oauth"
	"github.com/datvtph41107/bdspro/routers/api/v1/org"
	"github.com/datvtph41107/bdspro/routers/api/v1/user"
	weboauth "github.com/datvtph41107/bdspro/routers/web/oauth"
	appctx "github.com/datvtph41107/bdspro/services/context"
	security "github.com/datvtph41107/bdspro/services/security"
	"github.com/gin-gonic/gin"
)

// Dependencies are process-scoped HTTP capabilities. cmd and routers/init.go
// construct them; this package owns registration, not external resource lifetime.
type Dependencies struct {
	Auth          *user.AuthHandler
	OAuth2        *user.OAuth2Handler
	Authorization *oauth.Handler
	Browser       *weboauth.Handler
	PhoneEnabled  bool
	TokenMaker    authcrypto.Maker
	IPActive      *ipactive.Middleware
	Store         db.Store
	InviteKey     string
}

// orgAssignment loads the resource visible through the caller's active
// Account -> Person -> Organization membership. It is not write authority.
func orgAssignment(denied error) func(*appctx.APIContext) {
	return func(ctx *appctx.APIContext) {
		actor, ok := ctx.VerifiedActor()
		if !ok {
			ctx.AuthenticationRequired()
			return
		}
		id, err := ctx.PositivePathID("id")
		if err != nil {
			ctx.APIErrorAuto(organization_model.ErrInvalidOrganization)
			return
		}
		store, err := db.StoreFromContext(ctx)
		if err != nil {
			ctx.APIErrorAuto(err)
			return
		}
		access, err := organization_model.GetForAccount(ctx, store, actor.AccountID, id)
		if errors.Is(err, organization_model.ErrOrganizationNotFound) {
			if denied != nil {
				// Preserve the owner write endpoint's existing public error.
				ctx.APIErrorAuto(denied)
			} else {
				ctx.APIErrorAuto(organization_model.ErrOrganizationNotFound)
			}
			return
		}
		if err != nil {
			ctx.APIErrorAuto(err)
			return
		}
		ctx.Org = &appctx.APIOrganization{
			Organization: access.OrganizationModel(),
			Access:       &access,
		}
	}
}

// teamAssignment resolves a Team from its own route rather than treating
// Team ID as Organization ID. The SQLC read is account-access scoped.
func teamAssignment() func(*appctx.APIContext) {
	return func(ctx *appctx.APIContext) {
		actor, ok := ctx.VerifiedActor()
		if !ok {
			ctx.AuthenticationRequired()
			return
		}
		id, err := ctx.PositivePathID("team_id")
		if err != nil {
			ctx.APIInvalid(err)
			return
		}
		store, err := db.StoreFromContext(ctx)
		if err != nil {
			ctx.APIErrorAuto(err)
			return
		}
		team, err := organization_model.GetTeam(ctx, store, actor.AccountID, id)
		if err != nil {
			ctx.APIErrorAuto(err)
			return
		}
		access, err := organization_model.GetForAccount(ctx, store, actor.AccountID, team.OrgID)
		if err != nil {
			ctx.APIErrorAuto(err)
			return
		}
		ctx.Org = &appctx.APIOrganization{
			Organization: access.OrganizationModel(),
			Team:         team,
			Access:       &access,
		}
	}
}

// reqToken verifies the result of upstream authentication/session middleware.
// JWT verification is not duplicated here.
func reqToken() func(*appctx.APIContext) {
	return func(ctx *appctx.APIContext) {
		if _, ok := ctx.VerifiedActor(); ok {
			return
		}
		ctx.AuthenticationRequired()
	}
}

func reqOrgMembership() func(*appctx.APIContext) {
	return func(ctx *appctx.APIContext) {
		if ctx.Org == nil || ctx.Org.Organization == nil || ctx.Org.Access == nil {
			ctx.APIErrorAuto(organization_model.ErrOrganizationNotFound)
		}
	}
}

// Only a preflight permission check. SQLC owns commit-order authority.
func reqOrgOwnership(denied ...error) func(*appctx.APIContext) {
	ownerError := error(organization_model.ErrTeamOwnerRequired)
	if len(denied) > 0 && denied[0] != nil {
		ownerError = denied[0]
	}
	return func(ctx *appctx.APIContext) {
		if ctx.Org == nil || ctx.Org.Organization == nil || ctx.Org.Access == nil {
			ctx.APIErrorAuto(organization_model.ErrOrganizationNotFound)
			return
		}
		if ctx.Org.Access.Role != organization_model.RoleOwner {
			ctx.APIErrorAuto(ownerError)
		}
	}
}

// bind validates once and saves a typed, per-request form for the action.
// bind owns the public API's validation and typed form publication. Gitea's
// API binder likewise calls modules/web/middleware and then web.SetForm.
func bind[T any](_ T) func(*appctx.APIContext) {
	return func(ctx *appctx.APIContext) {
		form, err := middleware.BindJSON[T](ctx)
		if err != nil {
			ctx.Validation(err, form)
			return
		}
		web.SetForm(ctx, form)
	}
}

// Routes owns all API method/path/group declarations. A single Gin engine
// executes all middleware. web.Router converts typed request actions at
// registration; no second router, registry, or handler dispatch exists.
func Routes(root *gin.Engine, d Dependencies) *web.Router {
	// Gin applies APIContexter after route matching, before typed actions.
	// It is the sole request-local APIContext owner.
	root.Use(appctx.APIContexter())
	m := web.NewRouter(root)
	m.Get("/health", Health)
	registerPublicRoutes(m, d)
	return m
}

func registerPublicRoutes(m *web.Router, d Dependencies) {
	// GinActions is the explicit adapter for existing Gin protocol handlers.
	// New API actions use func(*context.APIContext) directly.
	m.Post("/users", appctx.GinActions(
		d.IPActive.For(ipactive.Register), d.Auth.Register,
	)...)
	m.Post("/users/login", appctx.GinActions(
		security.LogLoginResponse(), d.IPActive.For(ipactive.Login), d.Auth.Login,
	)...)
	m.Get("/users/verify_email", appctx.GinActions(d.Auth.VerifyEmailPage)...)
	m.Post("/users/verify_email", appctx.GinActions(
		d.IPActive.For(ipactive.VerifyEmail), d.Auth.VerifyEmail,
	)...)
	m.Post("/users/verify_email/resend", appctx.GinActions(
		d.IPActive.For(ipactive.ResendVerificationEmail), d.Auth.ResendVerificationEmail,
	)...)
	m.Post("/tokens/renew_access", appctx.GinActions(
		d.IPActive.For(ipactive.RenewAccess), d.Auth.Renew,
	)...)
	m.Post("/tokens/revoke", appctx.GinActions(
		d.IPActive.For(ipactive.RevokeToken), d.Auth.Revoke,
	)...)
	m.Post("/users/password/reset/request", appctx.GinActions(
		d.IPActive.For(ipactive.RequestPasswordReset), d.Auth.RequestPasswordReset,
	)...)
	m.Get("/users/password/reset", appctx.GinActions(d.Auth.PasswordResetPage)...)
	m.Post("/users/password/reset", appctx.GinActions(
		d.IPActive.For(ipactive.ConfirmPasswordReset), d.Auth.ResetPassword,
	)...)

	if d.Authorization != nil {
		preview := d.Authorization.AuthorizePreview
		approve := d.Authorization.Authorize
		authenticate := d.Authorization.Authenticate(d.TokenMaker)
		if d.Browser != nil {
			preview = d.Browser.AuthorizePreview
			approve = d.Browser.Authorize
			authenticate = d.Browser.Authenticate
			m.Get("/oauth/login", appctx.GinActions(d.Browser.LoginPage)...)
			loginHandlers := []gin.HandlerFunc{d.Browser.Login}
			if d.IPActive != nil {
				loginHandlers = append([]gin.HandlerFunc{d.IPActive.For(ipactive.Login)}, loginHandlers...)
			}
			m.Post("/oauth/login", appctx.GinActions(loginHandlers...)...)
			m.Post("/oauth/logout", appctx.GinActions(
				d.Browser.RequireBrowserSession, d.Browser.Logout,
			)...)
		}
		m.Get("/oauth/authorize", appctx.GinActions(authenticate, preview)...)
		m.Post("/oauth/authorize", appctx.GinActions(authenticate, approve)...)
		// OAuth authorization-server tokens are NOT first-party login sessions.
		m.Post("/oauth/token", appctx.GinActions(
			d.IPActive.For(ipactive.RenewAccess), d.Authorization.Token,
		)...)
		m.Post("/oauth/revoke", appctx.GinActions(
			d.IPActive.For(ipactive.RevokeToken), d.Authorization.Revoke,
		)...)
		m.Get("/oauth/userinfo", appctx.GinActions(d.Authorization.UserInfo)...)
	}
	if d.OAuth2 != nil {
		if d.Browser != nil && d.Browser.OIDCEnabled {
			m.Get("/oauth/login/oidc/start", appctx.GinActions(
				d.IPActive.For(ipactive.StartOAuth2Login), d.OAuth2.StartBrowserLogin,
			)...)
		}
		m.Get("/users/oauth2/login/start", appctx.GinActions(
			d.IPActive.For(ipactive.StartOAuth2Login), d.OAuth2.StartLogin,
		)...)
		m.Get("/users/oauth2/callback", appctx.GinActions(d.OAuth2.Callback)...)
	}
	if d.PhoneEnabled {
		m.Post("/users/phone/register/start", appctx.GinActions(
			d.IPActive.For(ipactive.StartPhoneRegister), d.Auth.StartPhoneRegister,
		)...)
		m.Post("/users/phone/register/verify", appctx.GinActions(
			d.IPActive.For(ipactive.VerifyPhoneRegister), d.Auth.VerifyPhoneRegister,
		)...)
		m.Post("/users/phone/login/start", appctx.GinActions(
			d.IPActive.For(ipactive.StartPhoneLogin), d.Auth.StartPhoneLogin,
		)...)
		m.Post("/users/phone/login/verify", appctx.GinActions(
			d.IPActive.For(ipactive.VerifyPhoneLogin), d.Auth.VerifyPhoneLogin,
		)...)
	}

	// The live first-party PostgreSQL session is verified once at ingress.
	// Resource assignment and SQLC write authority remain separate concerns.
	m.Group("", func() {
		registerAuthenticatedRoutes(m, d)
	}, appctx.GinActions(
		appctx.Authenticate(d.TokenMaker), appctx.RequireSession(d.Store),
	)...)
}

func registerAuthenticatedRoutes(m *web.Router, d Dependencies) {
	if d.Authorization != nil {
		m.Get("/oauth/clients", appctx.GinActions(d.Authorization.ListClients)...)
		m.Post("/oauth/clients", appctx.GinActions(
			appctx.VerifiedContact(d.Store), d.Authorization.CreateClient,
		)...)
		m.Delete("/oauth/clients/:id", appctx.GinActions(d.Authorization.DeleteClient)...)
		m.Get("/users/me/oauth/authorizations", appctx.GinActions(d.Authorization.ListAuthorizations)...)
		m.Delete("/users/me/oauth/authorizations/:client_id", appctx.GinActions(d.Authorization.RevokeAuthorization)...)
	}
	m.Get("/users/me", user.GetAuthenticatedUser)
	m.Patch("/users/me/profile", bind(api.ProfileUpdateRequest{}), user.UpdateProfile)
	m.Get("/users/me/settings", user.GetUserSettings)
	m.Patch("/users/me/settings", bind(api.SettingsUpdateRequest{}), user.UpdateUserSettings)
	m.Get("/users/me/connections", user.GetConnections)
	m.Get("/users/me/sessions", appctx.GinActions(d.Auth.ListSessions)...)
	m.Delete("/users/me/sessions/:id", appctx.GinActions(d.Auth.RevokeOwnSession)...)
	m.Post("/users/me/sessions/revoke_all", appctx.GinActions(d.Auth.RevokeAllSessions)...)
	m.Post("/users/me/email/start", appctx.GinActions(
		d.IPActive.For(ipactive.StartEmailLink), d.Auth.StartEmailLink,
	)...)
	m.Post("/users/me/email/verify", appctx.GinActions(
		d.IPActive.For(ipactive.ConfirmEmailLink), d.Auth.ConfirmEmailLink,
	)...)

	// Team invitation still owns an immutable signing dependency, but all
	// route registration follows one typed Group pattern.
	m.Group("/orgs", func() {
		registerOrgRoutes(m)
	}, appctx.GinActions(appctx.VerifiedContact(d.Store))...)
	m.Group("/teams", func() {
		registerTeamRoutes(m, d)
	}, appctx.GinActions(appctx.VerifiedContact(d.Store))...)

	if d.OAuth2 != nil {
		m.Post("/users/me/oauth2/start", appctx.GinActions(
			d.IPActive.For(ipactive.StartOAuth2Link), d.OAuth2.StartLink,
		)...)
	}
	if d.OAuth2 != nil && d.PhoneEnabled {
		m.Post("/users/me/phone/oauth2/start", appctx.GinActions(
			appctx.VerifiedEmail(d.Store), d.IPActive.For(ipactive.StartOAuth2PhoneBind), d.OAuth2.StartPhoneBind,
		)...)
	}
	if d.PhoneEnabled {
		m.Group("/users/me/phone", func() {
			registerPhoneBindRoutes(m, d)
		}, appctx.GinActions(appctx.VerifiedEmail(d.Store))...)
	}
}

func registerOrgRoutes(m *web.Router) {
	m.Get("", org.List)
	m.Get("/:id/members", org.Members)
	m.Post("/:id/members", org.AddMember)
	m.Delete("/:id/members/:person_id", org.RemoveMember)
	m.Put("/:id/owners/:person_id", org.PromoteOwner)
	m.Delete("/:id/owners/:person_id", org.DemoteOwner)
	m.Post("/:id/owners/transfer", org.TransferOwner)
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

func registerTeamRoutes(m *web.Router, d Dependencies) {
	m.Post("/invites/accept", append(
		appctx.GinActions(d.IPActive.For(ipactive.AcceptTeamInvite)),
		bind(api.AcceptTeamInviteOption{}), org.AcceptInvite,
	)...)
	m.Group("/:team_id", func() {
		// Keep assignment on GET only: mutations still authorize with live
		// acting-session and owner-scoped SQLC checks in services/org.
		m.Get("", teamAssignment(), reqToken(), org.GetTeam)
		m.Patch("", bind(api.RenameTeamOption{}), org.EditTeam)
		m.Get("/members", reqToken(), org.ListTeamMembers)
		m.Post("/members", reqToken(), bind(api.AddTeamUserOption{}), org.AddTeamMember)
		m.Delete("/members/:person_id", reqToken(), org.RemoveTeamMember)
		m.Post("/invites", append(
			appctx.GinActions(d.IPActive.For(ipactive.CreateTeamInvite)),
			bind(api.CreateTeamInviteOption{}), org.CreateInvite(d.InviteKey),
		)...)
		m.Delete("/invites/:invite_id", org.RevokeInvite)
	})
}

func registerPhoneBindRoutes(m *web.Router, d Dependencies) {
	m.Post("/start", appctx.GinActions(
		d.IPActive.For(ipactive.StartPhoneBind), d.Auth.StartPhoneBind,
	)...)
	m.Post("/verify", appctx.GinActions(
		d.IPActive.For(ipactive.VerifyPhoneBind), d.Auth.VerifyPhoneBind,
	)...)
}

// @Summary Check API health
// @Tags System
// @Success 200 "Service is healthy"
// @Router /health [get]
func Health(ctx *appctx.APIContext) {
	ctx.Status(http.StatusOK)
}
