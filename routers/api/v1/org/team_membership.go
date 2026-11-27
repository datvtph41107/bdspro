package org

import (
	"errors"
	"net/http"

	"github.com/datvtph41107/bdspro/modules/structs"
	"github.com/datvtph41107/bdspro/modules/web"
	"github.com/datvtph41107/bdspro/routers/api/v1/utils"
	appctx "github.com/datvtph41107/bdspro/services/context"
	"github.com/datvtph41107/bdspro/services/mailer"
	orgservice "github.com/datvtph41107/bdspro/services/org"
	"github.com/google/uuid"
)

func teamFromPath(ctx *appctx.APIContext) (int64, bool) {
	id, err := ctx.PositivePathID("team")
	if err != nil {
		ctx.APIInvalid(err)
		return 0, false
	}
	return id, true
}

func ListTeamMembers(ctx *appctx.APIContext) {
	id, ok := teamFromPath(ctx)
	if !ok {
		return
	}
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
	r, err := orgservice.ListTeamMembers(ctx, *ctx.Doer, ctx.Org.Organization.ID, id, opts)
	if err != nil {
		writeOrganizationResult(ctx, http.StatusOK, nil, err)
		return
	}
	ctx.SetLinkHeader(r.Total, opts.Page, opts.PageSize)
	ctx.SetTotalCountHeader(r.Total)
	ctx.JSON(http.StatusOK, map[string]any{"items": r.Items, "page": opts.Page, "limit": opts.PageSize, "total": r.Total})
}

func GetOwnTeamMembership(ctx *appctx.APIContext) {
	id, ok := teamFromPath(ctx)
	if !ok {
		return
	}
	r, err := orgservice.GetOwnTeamMembership(ctx, ctx.Doer.AccountID, ctx.Org.Organization.ID, id)
	writeOrganizationResult(ctx, http.StatusOK, r, err)
}

func AddTeamMember(ctx *appctx.APIContext) {
	id, ok := teamFromPath(ctx)
	if !ok {
		return
	}
	form := web.GetForm[*structs.AddOrganizationMemberOption](ctx)
	person, ok := personFromForm(ctx, form.PersonID)
	if !ok {
		return
	}
	r, err := orgservice.AddTeamMember(ctx, *ctx.Doer, ctx.Org.Organization.ID, id, person)
	writeOrganizationResult(ctx, http.StatusCreated, r, err)
}

func RemoveTeamMember(ctx *appctx.APIContext) {
	id, ok := teamFromPath(ctx)
	if !ok {
		return
	}
	person, err := ctx.PositivePathID("person")
	if err != nil {
		ctx.APIInvalid(err)
		return
	}
	err = orgservice.RemoveTeamMember(ctx, *ctx.Doer, ctx.Org.Organization.ID, id, person)
	writeOrganizationResult(ctx, http.StatusNoContent, nil, err)
}

func CreateTeamInvitation(ctx *appctx.APIContext) {
	ctx.Header("Cache-Control", "no-store")
	id, ok := teamFromPath(ctx)
	if !ok {
		return
	}
	form := web.GetForm[*structs.CreateTeamInviteOption](ctx)
	r, err := orgservice.CreateTeamInvitation(ctx, *ctx.Doer, ctx.Org.Organization.ID, id, form.Recipient, form.ExpiresIn)
	if errors.Is(err, mailer.ErrUnavailable) {
		ctx.APIError(http.StatusServiceUnavailable, "service_unavailable", "invitation delivery unavailable")
		return
	}
	writeOrganizationResult(ctx, http.StatusCreated, r, err)
}

func RevokeTeamInvitation(ctx *appctx.APIContext) {
	id, ok := teamFromPath(ctx)
	if !ok {
		return
	}
	raw := ctx.CurrentRequest().PathValue("invite")
	invite, err := uuid.Parse(raw)
	if err != nil || invite == uuid.Nil || invite.String() != raw {
		ctx.FieldError("invite", "invalid_id", "invite must be a UUID")
		return
	}
	err = orgservice.RevokeTeamInvitation(ctx, *ctx.Doer, ctx.Org.Organization.ID, id, invite)
	writeOrganizationResult(ctx, http.StatusNoContent, nil, err)
}

func AcceptTeamInvitation(ctx *appctx.APIContext) {
	ctx.Header("Cache-Control", "no-store")
	form := web.GetForm[*structs.AcceptTeamInviteOption](ctx)
	r, err := orgservice.AcceptTeamInvitation(ctx, *ctx.Doer, form.InviteToken)
	writeOrganizationResult(ctx, http.StatusOK, r, err)
}
