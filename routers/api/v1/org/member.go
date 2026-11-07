package org

import (
	"net/http"

	organization_model "github.com/datvtph41107/bdspro/models/organization"
	"github.com/datvtph41107/bdspro/modules/structs"
	appctx "github.com/datvtph41107/bdspro/services/context"
	"github.com/datvtph41107/bdspro/services/convert"
)

func (h *Handler) Members(c *appctx.APIContext) {
	id, ok := organizationID(c)
	if !ok {
		return
	}
	members, err := organization_model.GetOrgMembers(c.Request.Context(), h.store, c.Doer.AccountID, id)
	if err != nil {
		c.APIErrorAuto(err)
		return
	}
	c.JSON(http.StatusOK, convert.ToActiveOrganizationMembers(members))
}

func (h *Handler) AddMember(c *appctx.APIContext) {
	id, ok := organizationID(c)
	if !ok {
		return
	}
	var req structs.AddOrgUserOption
	if err := c.ShouldBindJSON(&req); err != nil {
		appctx.Validation(c.Context, err, &req)
		return
	}
	member, err := organization_model.AddOrgUser(c.Request.Context(), h.store, c.Doer.AccountID, id, req.PersonID)
	if err != nil {
		c.APIErrorAuto(err)
		return
	}
	c.JSON(http.StatusCreated, convert.ToOrganizationMember(member))
}
