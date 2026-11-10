package apiv1

import (
	"errors"
	"net/http"

	db "github.com/datvtph41107/bdspro/models/db"
	organization_model "github.com/datvtph41107/bdspro/models/organization"
	"github.com/datvtph41107/bdspro/modules/ipactive"
	api "github.com/datvtph41107/bdspro/modules/structs"
	"github.com/datvtph41107/bdspro/modules/web"
	"github.com/datvtph41107/bdspro/modules/web/middleware"
	"github.com/datvtph41107/bdspro/routers/api/v1/org"
	"github.com/datvtph41107/bdspro/routers/api/v1/oauth"
	"github.com/datvtph41107/bdspro/routers/api/v1/user"
	appctx "github.com/datvtph41107/bdspro/services/context"
	security "github.com/datvtph41107/bdspro/services/security"
	"github.com/gin-gonic/gin"
)

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

// Routes returns an independently constructed API subrouter, following
// Gitea's route composition contract. Capabilities are initialized at startup.
func Routes() *web.Router {
	d := getRuntime()
	m := web.NewRouter()
	m.AfterRouting(appctx.APIContexter())
	m.Get("/health", Health)

	// The v1 route tree owns group declarations. Bind process-scoped capabilities
	// only at registration; request-scoped actor/resources belong to APIContext.
	// GinActions is the explicit adapter for existing Gin protocol handlers.
	// New API actions use func(*context.APIContext) directly.
	m.Post("/users",
		security.IPActiveFor(ipactive.Register),
		user.Register,
	)
	m.Post("/users/login",
		security.LogLoginResponse(),
		security.IPActiveFor(ipactive.Login),
		user.Login,
	)
	m.Get("/users/verify_email", user.VerifyEmailPage)
	m.Post("/users/verify_email", security.IPActiveFor(ipactive.VerifyEmail), user.VerifyEmail)
	m.Post("/users/verify_email/resend",
		security.IPActiveFor(ipactive.ResendVerificationEmail),
		user.ResendVerificationEmail,
	)
	m.Post("/tokens/renew_access",
		security.IPActiveFor(ipactive.RenewAccess),
		user.Renew,
	)
	m.Post("/tokens/revoke",
		security.IPActiveFor(ipactive.RevokeToken),
		user.Revoke,
	)
	m.Post("/users/password/reset/request",
		security.IPActiveFor(ipactive.RequestPasswordReset),
		user.RequestPasswordReset,
	)
	m.Get("/users/password/reset", user.PasswordResetPage)
	m.Post("/users/password/reset", security.IPActiveFor(ipactive.ConfirmPasswordReset), user.ResetPassword)

	if d.AuthorizationEnabled {
		if d.Browser != nil {
			m.Get("/oauth/login", appctx.GinActions(d.Browser.LoginPage)...)
			loginHandlers := []gin.HandlerFunc{d.Browser.Login}
			if d.IPActive != nil {
				loginHandlers = append([]gin.HandlerFunc{d.IPActive.For(ipactive.Login)}, loginHandlers...)
			}
			m.Post("/oauth/login", appctx.GinActions(loginHandlers...)...)
			m.Post("/oauth/logout", appctx.GinActions(d.Browser.RequireBrowserSession, d.Browser.Logout)...)
			// HTML consent and API bearer authorization share the same typed
			// OAuth service actions. Browser middleware installs the actor.
			m.Get("/oauth/authorize", appctx.GinActions(d.Browser.Authenticate, d.Browser.AuthorizePreview)...)
			m.Post("/oauth/authorize", appctx.GinActions(d.Browser.Authenticate, d.Browser.Authorize)...)
		} else {
			m.Get("/oauth/authorize", appctx.Authenticate(), appctx.RequireSession(), oauth.AuthorizePreview)
			m.Post("/oauth/authorize", appctx.Authenticate(), appctx.RequireSession(), oauth.Authorize)
		}
		// OAuth authorization-server tokens have an independent lifecycle.
		m.Post("/oauth/token", security.IPActiveFor(ipactive.RenewAccess), oauth.Token)
		m.Post("/oauth/revoke", security.IPActiveFor(ipactive.RevokeToken), oauth.Revoke)
		m.Get("/oauth/userinfo", oauth.UserInfo)
	}

	if d.OAuth2Enabled {
		if d.Browser != nil && d.Browser.OIDCEnabled {
			m.Get("/oauth/login/oidc/start",
				security.IPActiveFor(ipactive.StartOAuth2Login), user.StartBrowserLogin)
		}
		m.Get("/users/oauth2/login/start",
			security.IPActiveFor(ipactive.StartOAuth2Login), user.StartLogin)
		m.Get("/users/oauth2/callback", user.Callback)
	}
	if d.PhoneEnabled {
		m.Post("/users/phone/register/start", security.IPActiveFor(ipactive.StartPhoneRegister), user.StartPhoneRegister)
		m.Post("/users/phone/register/verify", security.IPActiveFor(ipactive.VerifyPhoneRegister), user.VerifyPhoneRegister)
		m.Post("/users/phone/login/start", security.IPActiveFor(ipactive.StartPhoneLogin), user.StartPhoneLogin)
		m.Post("/users/phone/login/verify", security.IPActiveFor(ipactive.VerifyPhoneLogin), user.VerifyPhoneLogin)
	}

	// The live first-party PostgreSQL session is verified once at ingress.
	// Resource assignment and SQLC write authority remain separate concerns.
	m.Group("", func() {
		if d.AuthorizationEnabled {
			m.Get("/oauth/clients", oauth.ListClients)
			m.Post("/oauth/clients", appctx.VerifiedContact(), oauth.CreateClient)
			m.Delete("/oauth/clients/:id", oauth.DeleteClient)
			// First-party actor authority is verified by the parent group.
			m.Get("/users/me/oauth/authorizations", oauth.ListAuthorizations)
			m.Delete("/users/me/oauth/authorizations/:client_id", oauth.RevokeAuthorization)
		}
		m.Get("/users/me", user.GetAuthenticatedUser)
		m.Patch("/users/me/profile", bind(api.ProfileUpdateRequest{}), user.UpdateProfile)
		m.Get("/users/me/settings", user.GetUserSettings)
		m.Patch("/users/me/settings", bind(api.SettingsUpdateRequest{}), user.UpdateUserSettings)
		m.Get("/users/me/connections", user.GetConnections)
		m.Get("/users/me/sessions", user.ListSessions)
		m.Delete("/users/me/sessions/:id", user.RevokeOwnSession)
		m.Post("/users/me/sessions/revoke_all", user.RevokeAllSessions)
		m.Post("/users/me/email/start", security.IPActiveFor(ipactive.StartEmailLink), user.StartEmailLink)
		m.Post("/users/me/email/verify", security.IPActiveFor(ipactive.ConfirmEmailLink), user.ConfirmEmailLink)

		// Team invitation still owns an immutable signing dependency, but all
		// route registration follows one typed Group pattern.
		m.Group("/orgs", func() {
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
		}, appctx.VerifiedContact())
		m.Group("/teams", func() {
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
		}, appctx.VerifiedContact())

		if d.OAuth2Enabled {
			m.Post("/users/me/oauth2/start",
				security.IPActiveFor(ipactive.StartOAuth2Link), user.StartLink)
		}
		if d.OAuth2Enabled && d.PhoneEnabled {
			m.Post("/users/me/phone/oauth2/start",
				appctx.VerifiedEmail(),
				security.IPActiveFor(ipactive.StartOAuth2PhoneBind),
				user.StartOAuth2PhoneBind)
		}
		if d.PhoneEnabled {
			m.Group("/users/me/phone", func() {
				m.Post("/start", security.IPActiveFor(ipactive.StartPhoneBind), user.StartPhoneBind)
				m.Post("/verify", security.IPActiveFor(ipactive.VerifyPhoneBind), user.VerifyPhoneBind)
			}, appctx.VerifiedEmail())
		}
	}, appctx.Authenticate(), appctx.RequireSession())
	return m
}

// @Summary Check API health
// @Tags System
// @Success 200 "Service is healthy"
// @Router /health [get]
func Health(ctx *appctx.APIContext) {
	ctx.Status(http.StatusOK)
}
