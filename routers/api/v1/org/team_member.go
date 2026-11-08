package org

import (
	"net/http"
	organization_model "github.com/datvtph41107/bdspro/models/organization"
	appctx "github.com/datvtph41107/bdspro/services/context"
	"github.com/datvtph41107/bdspro/services/convert"
)

// Members lists only people with active organization and team membership.
func (h *TeamHandler) Members(c *appctx.APIContext) {
	teamID, ok := positivePathID(c, "team_id")
	if !ok {
		return
	}
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		c.APIInvalid(err)
		return
	}
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if err != nil || limit < 1 {
		c.APIInvalid(err)
		return
	}
	members, err := organization_model.ListTeamUsers(
		c.Request.Context(), h.store, c.Doer.AccountID, teamID,
		db.NewListOptions(page, limit, 20, 100),
	)
	if err != nil {
		c.APIErrorAuto(err)
		return
	}
	c.JSON(http.StatusOK, convert.ToTeamMembers(members))
}
