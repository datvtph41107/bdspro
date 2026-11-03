package org

import (
	"net/http"

	organization_model "github.com/datvtph41107/bdspro/models/organization"
	"github.com/datvtph41107/bdspro/modules/structs"
	"github.com/datvtph41107/bdspro/routers/api/middleware"
	"github.com/datvtph41107/bdspro/routers/api/response"
	"github.com/datvtph41107/bdspro/services/convert"
	"github.com/gin-gonic/gin"
)

func (h *Handler) Members(c *gin.Context) {
	id, ok := organizationID(c)
	if !ok {
		return
	}
	members, err := organization_model.GetOrgMembers(c.Request.Context(), h.store, middleware.AccountID(c), id)
	if err != nil {
		response.Error(c, err)
		return
	}
	c.JSON(http.StatusOK, convert.ToActiveOrganizationMembers(members))
}

func (h *Handler) AddMember(c *gin.Context) {
	id, ok := organizationID(c)
	if !ok {
		return
	}
	var req structs.AddOrgUserOption
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Validation(c, err, &req)
		return
	}
	member, err := organization_model.AddOrgUser(c.Request.Context(), h.store, middleware.AccountID(c), id, req.PersonID)
	if err != nil {
		response.Error(c, err)
		return
	}
	c.JSON(http.StatusCreated, convert.ToOrganizationMember(member))
}
