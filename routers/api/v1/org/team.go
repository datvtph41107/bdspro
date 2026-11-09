package org

import (
	"net/http"
	"strconv"

	db "github.com/datvtph41107/bdspro/models/db"
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

// ListTeams uses the Organization assigned at the HTTP boundary. The SQLC
// list still filters by the verified Account, never by a client-supplied ID.
func ListTeams(ctx *context.APIContext) {
	options, ok := teamListOptions(ctx)
	if !ok {
		return
	}
	actor, ok := ctx.VerifiedActor()
	if !ok {
		ctx.AuthenticationRequired()
		return
	}
	store, err := db.StoreFromContext(ctx)
	if err != nil {
		ctx.APIErrorAuto(err)
		return
	}
	teams, err := organization.ListTeams(ctx, store, actor.AccountID, ctx.Org.Organization.ID, options)
	if err != nil {
		ctx.APIErrorAuto(err)
		return
	}
	ctx.JSON(http.StatusOK, convert.ToTeams(teams))
}

// teamListOptions owns pagination at the HTTP boundary, independent of SQLC
// transport. The model validates the resulting offset before querying.
func teamListOptions(ctx *context.APIContext) (db.ListOptions, bool) {
	page, err := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		ctx.APIInvalid(err)
		return db.ListOptions{}, false
	}
	limit, err := strconv.Atoi(ctx.DefaultQuery("limit", "20"))
	if err != nil || limit < 1 {
		ctx.APIInvalid(err)
		return db.ListOptions{}, false
	}
	return db.NewListOptions(page, limit, 20, 100), true
}
