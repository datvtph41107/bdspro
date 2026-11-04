package org

import (
	"net/http"
	"strconv"

	db "github.com/datvtph41107/bdspro/models/db"
	organization_model "github.com/datvtph41107/bdspro/models/organization"
	"github.com/datvtph41107/bdspro/modules/structs"
	"github.com/datvtph41107/bdspro/routers/api/middleware"
	"github.com/datvtph41107/bdspro/routers/api/response"
	"github.com/datvtph41107/bdspro/services/convert"
	"github.com/gin-gonic/gin"
)

// TeamHandler keeps HTTP parsing separate from Team persistence and permissions.
type TeamHandler struct {
	store db.Store
}

func NewTeamHandler(store db.Store) *TeamHandler {
	return &TeamHandler{store: store}
}

func (h *TeamHandler) Create(c *gin.Context) {
	orgID, ok := organizationID(c)
	if !ok {
		return
	}
	var req structs.CreateTeamOption
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Validation(c, err, &req)
		return
	}
	team, err := organization_model.CreateTeam(c.Request.Context(), h.store, organization_model.CreateTeamOptions{
		OrganizationID: orgID, ActorAccountID: middleware.AccountID(c),
		Name: req.Name, Description: req.Description,
	})
	if err != nil {
		response.Error(c, err)
		return
	}
	c.JSON(http.StatusCreated, convert.ToTeam(team))
}

func (h *TeamHandler) List(c *gin.Context) {
	orgID, ok := organizationID(c)
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
	opts := db.NewListOptions(page, limit, 20, 100)
	teams, err := organization_model.ListTeams(c.Request.Context(), h.store, middleware.AccountID(c), orgID, opts)
	if err != nil {
		response.Error(c, err)
		return
	}
	c.JSON(http.StatusOK, convert.ToTeams(teams))
}

func (h *TeamHandler) Get(c *gin.Context) {
	id, ok := positivePathID(c, "team_id")
	if !ok {
		return
	}
	team, err := organization_model.GetTeam(c.Request.Context(), h.store, middleware.AccountID(c), id)
	if err != nil {
		response.Error(c, err)
		return
	}
	c.JSON(http.StatusOK, convert.ToTeam(team))
}

func (h *TeamHandler) Edit(c *gin.Context) {
	id, ok := positivePathID(c, "team_id")
	if !ok {
		return
	}
	var req structs.RenameTeamOption
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Validation(c, err, &req)
		return
	}
	team, err := organization_model.RenameTeam(c.Request.Context(), h.store, middleware.AccountID(c), id, req.Name)
	if err != nil {
		response.Error(c, err)
		return
	}
	c.JSON(http.StatusOK, convert.ToTeam(team))
}

func (h *TeamHandler) AddMember(c *gin.Context) {
	id, ok := positivePathID(c, "team_id")
	if !ok {
		return
	}
	var req structs.AddTeamUserOption
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Validation(c, err, &req)
		return
	}
	member, err := organization_model.AddTeamUser(c.Request.Context(), h.store, middleware.AccountID(c), id, req.PersonID)
	if err != nil {
		response.Error(c, err)
		return
	}
	c.JSON(http.StatusCreated, convert.ToTeamMember(member))
}

func (h *TeamHandler) RemoveMember(c *gin.Context) {
	teamID, ok := positivePathID(c, "team_id")
	if !ok {
		return
	}
	personID, ok := positivePathID(c, "person_id")
	if !ok {
		return
	}
	if err := organization_model.RemoveTeamUser(c.Request.Context(), h.store, middleware.AccountID(c), teamID, personID); err != nil {
		response.Error(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func positivePathID(c *gin.Context, key string) (int64, bool) {
	id, err := strconv.ParseInt(c.Param(key), 10, 64)
	if err != nil || id <= 0 {
		response.Invalid(c, err)
		return 0, false
	}
	return id, true
}
