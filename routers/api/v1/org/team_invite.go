package org

import (
	"net/http"
	"time"

	"github.com/datvtph41107/bdspro/modules/structs"
	appctx "github.com/datvtph41107/bdspro/services/context"
	"github.com/datvtph41107/bdspro/services/convert"
	"github.com/google/uuid"
)

// CreateInvite atomically persists the hashed proof and a durable Asynq intent.
// The HTTP response never returns the secret; it is delivered by SMTP.
func (h *TeamHandler) CreateInvite(c *appctx.APIContext) {
	teamID, ok := positivePathID(c, "team_id")
	if !ok {
		return
	}
	var req structs.CreateTeamInviteOption
	if err := c.ShouldBindJSON(&req); err != nil {
		appctx.Validation(c.Context, err, &req)
		return
	}
	created, err := h.team.CreateTeamInvite(
		c.Request.Context(), c.Doer.AccountID, c.Doer.SessionID, teamID,
		req.Email, 7*24*time.Hour, h.signingKey,
	)
	if err != nil {
		c.APIErrorAuto(err)
		return
	}
	c.JSON(http.StatusAccepted, structs.IssuedTeamInvite{
		ID: created.Invite.ID, TeamID: created.Invite.TeamID,
		InvitedEmail: created.Invite.InvitedEmail, ExpiresAt: created.Invite.ExpiresAt,
	})
}

// AcceptInvite requires an authenticated, verified account whose current
// organization membership and email match the invitation's SQL predicates.
func (h *TeamHandler) AcceptInvite(c *appctx.APIContext) {
	var req structs.AcceptTeamInviteOption
	if err := c.ShouldBindJSON(&req); err != nil {
		appctx.Validation(c.Context, err, &req)
		return
	}
	member, err := h.team.AcceptTeamInvite(c.Request.Context(), c.Doer.AccountID, c.Doer.SessionID, req.Token)
	if err != nil {
		c.APIErrorAuto(err)
		return
	}
	c.JSON(http.StatusCreated, convert.ToTeamMember(member))
}

func (h *TeamHandler) RevokeInvite(c *appctx.APIContext) {
	teamID, ok := positivePathID(c, "team_id")
	if !ok {
		return
	}
	inviteID, err := uuid.Parse(c.Param("invite_id"))
	if err != nil {
		c.APIInvalid(err)
		return
	}
	// RevokeTeamInvite checks that the actor owns the invitation's team.
	if err := h.team.RevokeTeamInvite(c.Request.Context(), c.Doer.AccountID, c.Doer.SessionID, teamID, inviteID); err != nil {
		c.APIErrorAuto(err)
		return
	}
	c.Status(http.StatusNoContent)
}
