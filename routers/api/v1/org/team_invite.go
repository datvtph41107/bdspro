package org

import (
	"net/http"
	"time"

	api "github.com/datvtph41107/bdspro/modules/structs"
	"github.com/datvtph41107/bdspro/modules/web"
	"github.com/datvtph41107/bdspro/services/context"
	"github.com/datvtph41107/bdspro/services/convert"
	org_service "github.com/datvtph41107/bdspro/services/org"
	"github.com/google/uuid"
)

// CreateInvite parses HTTP input and delegates transaction authority and
// invitation proof signing to services/org; this action never reads a secret.
func CreateInvite(ctx *context.APIContext) {
	teamID, err := ctx.PositivePathID("team_id")
	if err != nil {
		ctx.APIInvalid(err)
		return
	}
	form := web.GetForm[*api.CreateTeamInviteOption](ctx)
	created, err := org_service.CreateTeamInvite(
		ctx, teamID, form.Email, 7*24*time.Hour,
	)
	if err != nil {
		ctx.APIErrorAuto(err)
		return
	}
	// The raw proof is never returned to the HTTP client. The durable
	// outbox is written in the same transaction as invitation state.
	ctx.JSON(http.StatusAccepted, api.IssuedTeamInvite{
		ID:           created.Invite.ID,
		TeamID:       created.Invite.TeamID,
		InvitedEmail: created.Invite.InvitedEmail,
		ExpiresAt:    created.Invite.ExpiresAt,
	})
}

// AcceptInvite uses the form bound at the route boundary. The org service
// owns token consumption, membership creation and current-session authority.
func AcceptInvite(ctx *context.APIContext) {
	form := web.GetForm[*api.AcceptTeamInviteOption](ctx)
	member, err := org_service.AcceptTeamInvite(ctx, form.Token)
	if err != nil {
		ctx.APIErrorAuto(err)
		return
	}
	ctx.JSON(http.StatusCreated, convert.ToTeamMember(member))
}

// RevokeInvite keeps transport parsing and output in the router. Only the
// service may perform the owner-scoped transaction and irrevocable update.
func RevokeInvite(ctx *context.APIContext) {
	teamID, err := ctx.PositivePathID("team_id")
	if err != nil {
		ctx.APIInvalid(err)
		return
	}
	inviteID, err := uuid.Parse(ctx.PathParam("invite_id"))
	if err != nil {
		ctx.APIInvalid(err)
		return
	}
	if err := org_service.RevokeTeamInvite(ctx, teamID, inviteID); err != nil {
		ctx.APIErrorAuto(err)
		return
	}
	ctx.Status(http.StatusNoContent)
}
