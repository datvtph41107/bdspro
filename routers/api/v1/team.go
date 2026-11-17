package apiv1

import (
	"net/http"
	"strconv"

	"github.com/datvtph41107/bdspro/modules/web"
	appctx "github.com/datvtph41107/bdspro/services/context"
	orgservice "github.com/datvtph41107/bdspro/services/org"
)

type CreateTeamOption struct {
	Name        string `json:"name" binding:"required,min=1,max=100"`
	Description string `json:"description" binding:"max=2000"`
}

type RenameTeamOption struct {
	Name string `json:"name" binding:"required,min=1,max=100"`
}

// TeamRoutes is mounted under /api/v1 and registers Gitea-style scopes.
// Before any action, auth verifies the JWT and its backing DB session,
// then org assignment verifies active membership.
func TeamRoutes(r *web.Router) {
	r.Group("/orgs/{org}", func() {
		r.Get("/teams", ListTeams)
		r.Post("/teams", bind(CreateTeamOption{}), CreateTeam)
		r.Get("/teams/{team}", GetTeam)
		r.Patch("/teams/{team}", bind(RenameTeamOption{}), RenameTeam)
	}, appctx.AuthenticateAccessToken, appctx.AssignOrganization)
}

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
	items, err := orgservice.ListTeams(ctx, ctx.Actor.AccountID, ctx.Org.ID, offset, limit)
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
	team, err := orgservice.GetTeam(ctx, ctx.Actor.AccountID, ctx.Org.ID, teamID)
	if err != nil {
		ctx.APIErrorAuto(err)
		return
	}
	ctx.JSON(http.StatusOK, team)
}

func CreateTeam(ctx *appctx.APIContext) {
	form := web.GetForm[*CreateTeamOption](ctx)
	team, err := orgservice.CreateTeam(ctx, ctx.Actor.AccountID, ctx.Org.ID, form.Name, form.Description)
	if err != nil {
		ctx.APIErrorAuto(err)
		return
	}
	ctx.JSON(http.StatusCreated, team)
}

func RenameTeam(ctx *appctx.APIContext) {
	form := web.GetForm[*RenameTeamOption](ctx)
	teamID, err := ctx.PositivePathID("team")
	if err != nil {
		ctx.APIInvalid(err)
		return
	}
	team, err := orgservice.RenameTeam(ctx, ctx.Actor.AccountID, ctx.Org.ID, teamID, form.Name)
	if err != nil {
		ctx.APIErrorAuto(err)
		return
	}
	ctx.JSON(http.StatusOK, team)
}
