package org

import (
	"net/http"

	db "github.com/datvtph41107/bdspro/models/db"
	organization_model "github.com/datvtph41107/bdspro/models/organization"
	"github.com/datvtph41107/bdspro/modules/structs"
	appctx "github.com/datvtph41107/bdspro/services/context"
	"github.com/datvtph41107/bdspro/services/convert"
	org_service "github.com/datvtph41107/bdspro/services/organization"
)

// TeamHandler keeps HTTP parsing separate from Team persistence and permissions.
type TeamHandler struct {
	store      db.Store
	team       *org_service.TeamService
	signingKey string
}

func NewTeamHandler(store db.Store, signingKey string) *TeamHandler {
	return &TeamHandler{store: store, team: org_service.NewTeamService(store), signingKey: signingKey}
}

func (h *TeamHandler) Create(c *appctx.APIContext) {
	orgID, ok := organizationID(c)
	if !ok {
		return
	}
	var req structs.CreateTeamOption
	if err := c.BindJSON(&req); err != nil {
		c.Validation(err, &req)
		return
	}
	team, err := h.team.CreateTeam(c, c.Doer.SessionID, organization_model.CreateTeamOptions{
		OrganizationID: orgID, ActorAccountID: c.Doer.AccountID,
		Name: req.Name, Description: req.Description,
	})
	if err != nil {
		c.APIErrorAuto(err)
		return
	}
	c.JSON(http.StatusCreated, convert.ToTeam(team))
}

func (h *TeamHandler) Get(c *appctx.APIContext) {
	id, ok := positivePathID(c, "team_id")
	if !ok {
		return
	}
	team, err := organization_model.GetTeam(c, h.store, c.Doer.AccountID, id)
	if err != nil {
		c.APIErrorAuto(err)
		return
	}
	c.JSON(http.StatusOK, convert.ToTeam(team))
}

func (h *TeamHandler) Edit(c *appctx.APIContext) {
	id, ok := positivePathID(c, "team_id")
	if !ok {
		return
	}
	var req structs.RenameTeamOption
	if err := c.BindJSON(&req); err != nil {
		c.Validation(err, &req)
		return
	}
	team, err := h.team.RenameTeam(c, c.Doer.AccountID, c.Doer.SessionID, id, req.Name)
	if err != nil {
		c.APIErrorAuto(err)
		return
	}
	c.JSON(http.StatusOK, convert.ToTeam(team))
}

func (h *TeamHandler) AddMember(c *appctx.APIContext) {
	id, ok := positivePathID(c, "team_id")
	if !ok {
		return
	}
	var req structs.AddTeamUserOption
	if err := c.BindJSON(&req); err != nil {
		c.Validation(err, &req)
		return
	}
	member, err := h.team.AddTeamUser(c, c.Doer.AccountID, c.Doer.SessionID, id, req.PersonID)
	if err != nil {
		c.APIErrorAuto(err)
		return
	}
	c.JSON(http.StatusCreated, convert.ToTeamMember(member))
}

func (h *TeamHandler) RemoveMember(c *appctx.APIContext) {
	teamID, ok := positivePathID(c, "team_id")
	if !ok {
		return
	}
	personID, ok := positivePathID(c, "person_id")
	if !ok {
		return
	}
	if err := h.team.RemoveTeamUser(c, c.Doer.AccountID, c.Doer.SessionID, teamID, personID); err != nil {
		c.APIErrorAuto(err)
		return
	}
	c.Status(http.StatusNoContent)
}

func positivePathID(c *appctx.APIContext, key string) (int64, bool) {
	id, err := c.PositivePathID(key)
	if err != nil {
		c.APIInvalid(err)
		return 0, false
	}
	return id, true
}
