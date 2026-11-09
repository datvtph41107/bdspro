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
	opts, ok := teamListOptions(c)
	if !ok {
		return
	}
	members, err := organization_model.ListTeamUsers(
		c, h.store, c.Doer.AccountID, teamID, opts,
	)
	if err != nil {
		c.APIErrorAuto(err)
		return
	}
	c.JSON(http.StatusOK, convert.ToTeamMembers(members))
}
