package org

import (
	"net/http"

	"github.com/datvtph41107/bdspro/models/organization"
	api "github.com/datvtph41107/bdspro/modules/structs"
	"github.com/datvtph41107/bdspro/modules/web"
	"github.com/datvtph41107/bdspro/services/context"
	"github.com/datvtph41107/bdspro/services/convert"
	org_service "github.com/datvtph41107/bdspro/services/organization"
)

// CreateTeam is the Gitea-style standalone action; owner permission is
// checked again in the SQLC transaction, not trusted from context snapshot.
func CreateTeam(ctx *context.APIContext) {
	form := web.GetForm[*api.CreateTeamOption](ctx)
	team := &organization.Team{
		OrgID:       ctx.Org.Organization.ID,
		Name:        form.Name,
		Description: form.Description,
	}
	if err := org_service.NewTeam(ctx, team); err != nil {
		ctx.APIErrorAuto(err)
		return
	}
	ctx.JSON(http.StatusCreated, convert.ToTeamModel(team))
}

// GetTeam consumes the model assigned by teamAssignment.
func GetTeam(ctx *context.APIContext) {
	ctx.JSON(http.StatusOK, convert.ToTeamModel(ctx.Org.Team))
}
