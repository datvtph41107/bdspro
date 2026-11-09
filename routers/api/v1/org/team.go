package org

import (
	"net/http"
	"github.com/datvtph41107/bdspro/models/organization"
	api "github.com/datvtph41107/bdspro/modules/structs"
	"github.com/datvtph41107/bdspro/modules/web"
	"github.com/datvtph41107/bdspro/routers/api/v1/utils"
	"github.com/datvtph41107/bdspro/services/context"
	"github.com/datvtph41107/bdspro/services/convert"
	org_service "github.com/datvtph41107/bdspro/services/org"
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

// EditTeam follows Gitea's package-action pattern. The route owns typed
// binding; this action owns path parsing and output, never the transaction.
func EditTeam(ctx *context.APIContext) {
	teamID, err := ctx.PositivePathID("team_id")
	if err != nil {
		ctx.APIInvalid(err)
		return
	}
	form := web.GetForm[*api.RenameTeamOption](ctx)
	team := &organization.Team{ID: teamID, Name: form.Name}
	if err := org_service.UpdateTeam(ctx, team); err != nil {
		ctx.APIErrorAuto(err)
		return
	}
	ctx.JSON(http.StatusOK, convert.ToTeamModel(team))
}

// GetTeam consumes the model assigned by teamAssignment.
func GetTeam(ctx *context.APIContext) {
	ctx.JSON(http.StatusOK, convert.ToTeamModel(ctx.Org.Team))
}

// ListTeams uses the Organization assigned at the HTTP boundary. The SQLC
// list still filters by the verified Account, never by a client-supplied ID.
func ListTeams(ctx *context.APIContext) {
	options, err := utils.GetListOptions(ctx)
	if err != nil {
		ctx.APIInvalid(err)
		return
	}
	teams, err := organization.ListTeamsForDoer(ctx, ctx.Org.Organization.ID, options)
	if err != nil {
		ctx.APIErrorAuto(err)
		return
	}
	ctx.JSON(http.StatusOK, convert.ToTeams(teams))
}

