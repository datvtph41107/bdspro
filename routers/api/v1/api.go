package apiv1

import (
	"net/http"

	"github.com/datvtph41107/bdspro/modules/ipactive"
	"github.com/datvtph41107/bdspro/modules/structs"
	"github.com/datvtph41107/bdspro/modules/web"
	"github.com/datvtph41107/bdspro/modules/web/middleware"
	"github.com/datvtph41107/bdspro/routers/api/v1/auth"
	org "github.com/datvtph41107/bdspro/routers/api/v1/org"
	"github.com/datvtph41107/bdspro/routers/api/v1/user"
	"github.com/datvtph41107/bdspro/routers/common"
	appctx "github.com/datvtph41107/bdspro/services/context"
)

func Routes() *web.Router {
	m := web.NewRouter()
	m.AfterRouting(appctx.APIContexter())
	m.Get("/health", Health)
	m.Group("/auth", func() {
		m.Get("/google/start", reqIPAdmission(ipactive.StartOAuth2Login), auth.GoogleStart)
		m.Get("/google/callback", reqIPAdmission(ipactive.StartOAuth2Login), auth.GoogleCallback)
		m.Post("/google/result", auth.BrowserMutation, reqIPAdmission(ipactive.StartOAuth2Login), auth.GoogleResult)
		m.Post("/google/complete", auth.BrowserMutation, reqIPAdmission(ipactive.StartOAuth2Login), bind(structs.GoogleComplete{}), auth.GoogleComplete)
		m.Get("/phone/channels", auth.PhoneChannels)
		m.Post("/phone/start", auth.BrowserMutation, reqIPAdmission(ipactive.StartPhoneRegister), bind(structs.PhoneStart{}), auth.PhoneStart)
		m.Post("/phone/resend", auth.BrowserMutation, reqIPAdmission(ipactive.StartPhoneRegister), bind(structs.PhoneResend{}), auth.PhoneResend)
		m.Post("/phone/verify", auth.BrowserMutation, reqIPAdmission(ipactive.VerifyPhoneRegister), bind(structs.PhoneVerify{}), auth.PhoneVerify)
		m.Post("/phone/complete", auth.BrowserMutation, reqIPAdmission(ipactive.StartPhoneRegister), bind(structs.PhoneComplete{}), auth.PhoneComplete)
		m.Post("/register/start", auth.BrowserMutation, reqIPAdmission(ipactive.Register), bind(structs.RegistrationStart{}), auth.RegisterStart)
		m.Post("/register/resend", auth.BrowserMutation, reqIPAdmission(ipactive.ResendVerificationEmail), bind(structs.RegistrationResend{}), auth.RegisterResend)
		m.Post("/register/verify", auth.BrowserMutation, reqIPAdmission(ipactive.VerifyEmail), bind(structs.RegistrationVerify{}), auth.RegisterVerify)
		m.Post("/register/complete", auth.BrowserMutation, reqIPAdmission(ipactive.Register), bind(structs.RegistrationComplete{}), auth.RegisterComplete)
		m.Post("/login", auth.BrowserMutation, reqIPAdmission(ipactive.Login), bind(structs.LoginRequest{}), auth.Login)
		m.Post("/refresh", auth.BrowserMutation, reqIPAdmission(ipactive.RenewAccess), auth.Refresh)
		m.Post("/logout", auth.BrowserMutation, reqIPAdmission(ipactive.RevokeToken), auth.Logout)
		m.Post("/password/reset/request", auth.BrowserMutation, reqIPAdmission(ipactive.RequestPasswordReset), bind(structs.PasswordResetRequest{}), auth.RequestPasswordReset)
		m.Post("/password/reset/confirm", auth.BrowserMutation, reqIPAdmission(ipactive.ConfirmPasswordReset), bind(structs.PasswordResetConfirm{}), auth.ConfirmPasswordReset)
	})
	m.Get("/me", common.APIAuth, user.GetCurrentUser)
	m.Group("/me", func() {
		m.Post("/reauth/password", auth.BrowserMutation, reqIPAdmission(ipactive.Login), bind(structs.PasswordReauthRequest{}), auth.ReauthenticatePassword)
		m.Post("/reauth/email/start", auth.BrowserMutation, reqIPAdmission(ipactive.StartEmailLink), auth.RequestPasswordSetProof)
		m.Post("/password/set", auth.BrowserMutation, reqIPAdmission(ipactive.Login), bind(structs.PasswordSetRequest{}), auth.SetPassword)
		m.Post("/email/start", auth.BrowserMutation, reqIPAdmission(ipactive.StartEmailLink), bind(structs.EmailLinkStart{}), auth.StartEmailLink)
		m.Post("/email/verify", auth.BrowserMutation, reqIPAdmission(ipactive.ConfirmEmailLink), bind(structs.ContactLinkVerify{}), auth.VerifyContactLink("email", ipactive.ConfirmEmailLink))
		m.Post("/phone/start", auth.BrowserMutation, reqIPAdmission(ipactive.StartPhoneBind), bind(structs.PhoneLinkStart{}), auth.StartPhoneLink)
		m.Post("/phone/verify", auth.BrowserMutation, reqIPAdmission(ipactive.VerifyPhoneBind), bind(structs.ContactLinkVerify{}), auth.VerifyContactLink("phone", ipactive.VerifyPhoneBind))
		m.Delete("/email", auth.BrowserMutation, reqIPAdmission(ipactive.Login), bind(structs.RecentAuthRequest{}), auth.RemoveCredential("email"))
		m.Delete("/phone", auth.BrowserMutation, reqIPAdmission(ipactive.Login), bind(structs.RecentAuthRequest{}), auth.RemoveCredential("phone"))
		m.Post("/google/start", auth.BrowserMutation, reqIPAdmission(ipactive.StartOAuth2Link), bind(structs.RecentAuthRequest{}), auth.GoogleLinkStart)
		m.Post("/google/result", auth.BrowserMutation, reqIPAdmission(ipactive.StartOAuth2Link), auth.GoogleLinkResult)
		m.Post("/reauth/google/start", auth.BrowserMutation, reqIPAdmission(ipactive.StartOAuth2Link), bind(structs.GoogleReauthRequest{}), auth.GoogleReauthStart)
		m.Post("/reauth/google/result", auth.BrowserMutation, reqIPAdmission(ipactive.StartOAuth2Link), auth.GoogleReauthResult)
		m.Delete("/google", auth.BrowserMutation, reqIPAdmission(ipactive.Login), bind(structs.RecentAuthRequest{}), auth.RemoveCredential("google"))
		m.Delete("/password", auth.BrowserMutation, reqIPAdmission(ipactive.Login), bind(structs.RecentAuthRequest{}), auth.RemoveCredential("password"))
		m.Post("/sessions/revoke-all", auth.BrowserMutation, reqIPAdmission(ipactive.RevokeToken), bind(structs.RecentAuthRequest{}), auth.RevokeAllSessions)
		m.Post("/password/change", auth.BrowserMutation, reqIPAdmission(ipactive.Login), bind(structs.PasswordChangeRequest{}), auth.ChangePassword)
		m.Combo("/profile").
			Get(user.GetProfile).
			Patch(bind(structs.UpdateProfileOption{}), user.UpdateProfile)
		m.Combo("/settings").
			Get(user.GetSettings).
			Patch(bind(structs.UpdateAccountSettingsOption{}), user.UpdateSettings)
		m.Patch("/username", bind(structs.UpdateUsernameRequest{}), user.UpdateUsername)
		m.Get("/credentials", user.GetCredentials)
		m.Get("/connections", user.ListConnections)
		m.Get("/sessions", user.ListSessions)
		m.Delete("/sessions/{session}", user.RevokeSession)
	}, common.APIAuth)
	m.Post("/team-invites/accept", common.APIAuth, auth.BrowserMutation, reqIPAdmission(ipactive.AcceptTeamInvite), bind(structs.AcceptTeamInviteOption{}), org.AcceptTeamInvitation)
	m.Combo("/orgs").
		Get(common.APIAuth, org.ListOrganizations).
		Post(common.APIAuth, auth.BrowserMutation, bind(structs.CreateOrganizationOption{}), org.CreateOrganization)
	m.Group("/orgs/{org}", func() {
		m.Combo("").
			Get(org.GetOrganization).
			Patch(reqOrgOwner(), auth.BrowserMutation, bind(structs.UpdateOrganizationOption{}), org.UpdateOrganization)
		m.Combo("/members").
			Get(reqOrgOwner(), org.ListOrganizationMembers).
			Post(reqOrgOwner(), auth.BrowserMutation, bind(structs.AddOrganizationMemberOption{}), org.AddOrganizationMember)
		m.Delete("/members/{person}", reqOrgOwner(), auth.BrowserMutation, org.RemoveOrganizationMember)
		m.Post("/leave", auth.BrowserMutation, org.LeaveOrganization)
		m.Put("/owners/{person}", reqOrgOwner(), auth.BrowserMutation, org.PromoteOrganizationMember)
		m.Delete("/owners/{person}", reqOrgOwner(), auth.BrowserMutation, org.DemoteOrganizationOwner)
		m.Post("/owners/transfer", reqOrgOwner(), auth.BrowserMutation, bind(structs.TransferOrganizationOwnerOption{}), org.TransferOrganizationOwner)
		m.Combo("/teams").
			Get(org.ListTeams).
			Post(reqOrgOwner(), reqIPAdmission(ipactive.CreateTeam), bind(structs.CreateTeamOption{}), org.CreateTeam)
		m.Get("/teams/{team}/members/me", org.GetOwnTeamMembership)
		m.Combo("/teams/{team}/members").
			Get(reqOrgOwner(), org.ListTeamMembers).
			Post(reqOrgOwner(), auth.BrowserMutation, bind(structs.AddOrganizationMemberOption{}), org.AddTeamMember)
		m.Delete("/teams/{team}/members/{person}", reqOrgOwner(), auth.BrowserMutation, org.RemoveTeamMember)
		m.Post("/teams/{team}/invites", reqOrgOwner(), auth.BrowserMutation, reqIPAdmission(ipactive.CreateTeamInvite), bind(structs.CreateTeamInviteOption{}), org.CreateTeamInvitation)
		m.Delete("/teams/{team}/invites/{invite}", reqOrgOwner(), auth.BrowserMutation, org.RevokeTeamInvitation)
		m.Combo("/teams/{team}").
			Get(org.GetTeam).
			Patch(reqOrgOwner(), reqIPAdmission(ipactive.RenameTeam), bind(structs.RenameTeamOption{}), org.RenameTeam)
	}, common.APIAuth, orgAssignment)
	return m
}

func bind[T any](tmpl T) any {
	return func(ctx *appctx.APIContext) {
		form, err := middleware.BindJSON[T](ctx)
		if err != nil {
			ctx.Validation(err, form)
			return
		}
		web.SetForm(ctx, form)
	}
}

func Health(ctx *appctx.APIContext) {
	ctx.JSON(http.StatusOK, map[string]string{"status": "ok"})
}
