package org

import (
	"net/http"
	"strconv"

	db "github.com/datvtph41107/bdspro/models/db"
	organization_model "github.com/datvtph41107/bdspro/models/organization"
	"github.com/datvtph41107/bdspro/routers/api/middleware"
	"github.com/datvtph41107/bdspro/routers/api/response"
	"github.com/datvtph41107/bdspro/services/convert"
	"github.com/gin-gonic/gin"
)

// Members lists only people with active organization and team membership.
func (h *TeamHandler) Members(c *gin.Context) {
	teamID, ok := positivePathID(c, "team_id")
	if !ok {
		return
	}
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		response.Invalid(c, err)
		return
	}
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if err != nil || limit < 1 {
		response.Invalid(c, err)
		return
	}
	members, err := organization_model.ListTeamUsers(
		c.Request.Context(), h.store, middleware.AccountID(c), teamID,
		db.NewListOptions(page, limit, 20, 100),
	)
	if err != nil {
		response.Error(c, err)
		return
	}
	c.JSON(http.StatusOK, convert.ToTeamMembers(members))
}
