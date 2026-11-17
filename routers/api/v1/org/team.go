package org

import (
	"errors"
	"net/http"

	"github.com/datvtph41107/bdspro/modules/structs"
	"github.com/datvtph41107/bdspro/routers/api/v1/utils"

	"github.com/datvtph41107/bdspro/modules/web"
	appctx "github.com/datvtph41107/bdspro/services/context"
	orgservice "github.com/datvtph41107/bdspro/services/org"
)

func ListTeams(ctx *appctx.APIContext) {
	opts, err := utils.GetListOptions(ctx)
	if err != nil {
		var fieldErr *utils.ListOptionsError
		if errors.As(err, &fieldErr) {
			ctx.FieldError(fieldErr.Field, "invalid_range", fieldErr.Message)
			return
		}
		ctx.APIErrorInternal(err)
		return
	}
	result, err := orgservice.ListTeams(ctx, ctx.Doer.AccountID, ctx.Org.Organization.ID, opts)
	if err != nil {
		ctx.APIErrorAuto(err)
		return
	}
	ctx.SetLinkHeader(result.Total, opts.PageSize)
	ctx.SetTotalCountHeader(result.Total)
	ctx.JSON(http.StatusOK, map[string]any{
		"items": result.Items, "page": opts.Page,
		"limit": opts.PageSize, "total": result.Total,
	})
}

func GetTeam(ctx *appctx.APIContext) {
	teamID, err := ctx.PositivePathID("team")
	if err != nil {
		ctx.APIInvalid(err)
		return
	}
	team, err := orgservice.GetTeam(ctx, ctx.Doer.AccountID, ctx.Org.Organization.ID, teamID)
	if err != nil {
		ctx.APIErrorAuto(err)
		return
	}
	ctx.JSON(http.StatusOK, team)
}

func CreateTeam(ctx *appctx.APIContext) {
	form := web.GetForm[*structs.CreateTeamOption](ctx)
	team, err := orgservice.CreateTeam(ctx, ctx.Doer.AccountID, ctx.Org.Organization.ID, form.Name, form.Description)
	if err != nil {
		ctx.APIErrorAuto(err)
		return
	}
	ctx.JSON(http.StatusCreated, team)
}

func RenameTeam(ctx *appctx.APIContext) {
	form := web.GetForm[*structs.RenameTeamOption](ctx)
	teamID, err := ctx.PositivePathID("team")
	if err != nil {
		ctx.APIInvalid(err)
		return
	}
	team, err := orgservice.RenameTeam(ctx, ctx.Doer.AccountID, ctx.Org.Organization.ID, teamID, form.Name)
	if err != nil {
		ctx.APIErrorAuto(err)
		return
	}
	ctx.JSON(http.StatusOK, team)
}
