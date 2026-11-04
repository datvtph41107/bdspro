package org

import (
	"net/http"
	"time"

	organization_model "github.com/datvtph41107/bdspro/models/organization"
	"github.com/datvtph41107/bdspro/modules/structs"
	"github.com/datvtph41107/bdspro/routers/api/middleware"
	"github.com/datvtph41107/bdspro/routers/api/response"
	"github.com/datvtph41107/bdspro/services/convert"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// CreateInvite atomically persists the hashed proof and a durable Asynq intent.
// The HTTP response never returns the secret; it is delivered by SMTP.
func (h *TeamHandler) CreateInvite(c *gin.Context) {
	teamID, ok := positivePathID(c, "team_id")
	if !ok {
		return
	}
	var req structs.CreateTeamInviteOption
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Validation(c, err, &req)
		return
	}
	created, err := organization_model.CreateTeamInvite(
		c.Request.Context(), h.store, middleware.AccountID(c), teamID,
		req.Email, 7*24*time.Hour, h.signingKey,
	)
	if err != nil {
		response.Error(c, err)
		return
	}
	c.JSON(http.StatusAccepted, structs.IssuedTeamInvite{
		ID: created.Invite.ID, TeamID: created.Invite.TeamID,
		InvitedEmail: created.Invite.InvitedEmail, ExpiresAt: created.Invite.ExpiresAt,
	})
}

// AcceptInvite requires an authenticated, verified account whose current
// organization membership and email match the invitation's SQL predicates.
func (h *TeamHandler) AcceptInvite(c *gin.Context) {
	var req structs.AcceptTeamInviteOption
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Validation(c, err, &req)
		return
	}
	member, err := organization_model.AcceptTeamInvite(c.Request.Context(), h.store, middleware.AccountID(c), req.Token)
	if err != nil {
		response.Error(c, err)
		return
	}
	c.JSON(http.StatusCreated, convert.ToTeamMember(member))
}

func (h *TeamHandler) RevokeInvite(c *gin.Context) {
	_, ok := positivePathID(c, "team_id")
	if !ok {
		return
	}
	inviteID, err := uuid.Parse(c.Param("invite_id"))
	if err != nil {
		response.Invalid(c, err)
		return
	}
	// RevokeTeamInvite checks that the actor owns the invitation's team.
	if err := organization_model.RevokeTeamInvite(c.Request.Context(), h.store, middleware.AccountID(c), inviteID); err != nil {
		response.Error(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
