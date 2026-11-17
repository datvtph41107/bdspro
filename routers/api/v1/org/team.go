package org

import (
	"net/http"
	"strconv"

	"github.com/datvtph41107/bdspro/modules/structs"

	"github.com/datvtph41107/bdspro/modules/web"
	appctx "github.com/datvtph41107/bdspro/services/context"
	orgservice "github.com/datvtph41107/bdspro/services/org"
)

func ListTeams(ctx *appctx.APIContext) {
	offset, limit := int32(0), int32(20)
	if raw := ctx.DefaultQuery("offset", "0"); raw != "" {
		value, err := strconv.ParseInt(raw, 10, 32)
		if err != nil || value < 0 {
			ctx.FieldError("offset", "invalid_range", "offset must be a non-negative integer")
			return
		}
		offset = int32(value)
	}
	if raw := ctx.DefaultQuery("limit", "20"); raw != "" {
		value, err := strconv.ParseInt(raw, 10, 32)
		if err != nil || value < 1 || value > 50 {
			ctx.FieldError("limit", "invalid_range", "limit must be between 1 and 50")
			return
		}
		limit = int32(value)
	}
	items, err := orgservice.ListTeams(ctx, ctx.Doer.AccountID, ctx.Org.Organization.ID, offset, limit)
	if err != nil {
		ctx.APIErrorAuto(err)
		return
	}
	ctx.JSON(http.StatusOK, map[string]any{"items": items, "limit": limit, "offset": offset})
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
