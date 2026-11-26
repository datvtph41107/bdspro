package org

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/datvtph41107/bdspro/modules/structs"
	"github.com/datvtph41107/bdspro/modules/util"
	"github.com/datvtph41107/bdspro/modules/web"
	appctx "github.com/datvtph41107/bdspro/services/context"
	orgservice "github.com/datvtph41107/bdspro/services/org"
)

func writeOrganizationResult(ctx *appctx.APIContext, status int, result any, err error) {
	if errors.Is(err, util.ErrUnauthenticated) {
		ctx.AuthenticationRequired()
		return
	}
	if err != nil {
		ctx.APIErrorAuto(err)
		return
	}
	if status == http.StatusNoContent {
		ctx.Status(status)
		return
	}
	ctx.JSON(status, result)
}

func ListOrganizations(ctx *appctx.APIContext) {
	r, err := orgservice.ListOrganizations(ctx, ctx.Doer.AccountID)
	writeOrganizationResult(ctx, http.StatusOK, map[string]any{"items": r}, err)
}

func GetOrganization(ctx *appctx.APIContext) {
	r, err := orgservice.GetOrganizationDetail(ctx, ctx.Doer.AccountID, ctx.Org.Organization.ID)
	writeOrganizationResult(ctx, http.StatusOK, r, err)
}

func CreateOrganization(ctx *appctx.APIContext) {
	form := web.GetForm[*structs.CreateOrganizationOption](ctx)
	r, err := orgservice.CreateOrganization(ctx, *ctx.Doer, form.Name)
	writeOrganizationResult(ctx, http.StatusCreated, r, err)
}

func UpdateOrganization(ctx *appctx.APIContext) {
	form := web.GetForm[*structs.UpdateOrganizationOption](ctx)
	if !form.Name.Has() {
		ctx.FieldError("name", "required", "name is required")
		return
	}
	r, err := orgservice.UpdateOrganization(ctx, *ctx.Doer, ctx.Org.Organization.ID, form.Name.Value())
	writeOrganizationResult(ctx, http.StatusOK, r, err)
}

func ListOrganizationMembers(ctx *appctx.APIContext) {
	r, err := orgservice.ListOrganizationMembers(ctx, *ctx.Doer, ctx.Org.Organization.ID)
	writeOrganizationResult(ctx, http.StatusOK, map[string]any{"items": r}, err)
}

func personFromForm(ctx *appctx.APIContext, raw string) (int64, bool) {
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 || strconv.FormatInt(id, 10) != raw {
		ctx.FieldError("person_id", "invalid_id", "person_id must be a positive decimal string")
		return 0, false
	}
	return id, true
}

func AddOrganizationMember(ctx *appctx.APIContext) {
	form := web.GetForm[*structs.AddOrganizationMemberOption](ctx)
	person, ok := personFromForm(ctx, form.PersonID)
	if !ok {
		return
	}
	r, err := orgservice.AddOrganizationMember(ctx, *ctx.Doer, ctx.Org.Organization.ID, person)
	writeOrganizationResult(ctx, http.StatusCreated, r, err)
}

func RemoveOrganizationMember(ctx *appctx.APIContext) {
	person, err := ctx.PositivePathID("person")
	if err != nil {
		ctx.APIInvalid(err)
		return
	}
	err = orgservice.RemoveOrganizationMember(ctx, *ctx.Doer, ctx.Org.Organization.ID, person)
	writeOrganizationResult(ctx, http.StatusNoContent, nil, err)
}

func LeaveOrganization(ctx *appctx.APIContext) {
	err := orgservice.LeaveOrganization(ctx, *ctx.Doer, ctx.Org.Organization.ID)
	writeOrganizationResult(ctx, http.StatusNoContent, nil, err)
}

func PromoteOrganizationMember(ctx *appctx.APIContext) {
	person, err := ctx.PositivePathID("person")
	if err != nil {
		ctx.APIInvalid(err)
		return
	}
	r, err := orgservice.PromoteOrganizationMember(ctx, *ctx.Doer, ctx.Org.Organization.ID, person)
	writeOrganizationResult(ctx, http.StatusOK, r, err)
}

func DemoteOrganizationOwner(ctx *appctx.APIContext) {
	person, err := ctx.PositivePathID("person")
	if err != nil {
		ctx.APIInvalid(err)
		return
	}
	err = orgservice.DemoteOrganizationOwner(ctx, *ctx.Doer, ctx.Org.Organization.ID, person)
	writeOrganizationResult(ctx, http.StatusNoContent, nil, err)
}

func TransferOrganizationOwner(ctx *appctx.APIContext) {
	form := web.GetForm[*structs.TransferOrganizationOwnerOption](ctx)
	person, ok := personFromForm(ctx, form.PersonID)
	if !ok {
		return
	}
	err := orgservice.TransferOrganizationOwner(ctx, *ctx.Doer, ctx.Org.Organization.ID, person, form.ReauthenticationProof)
	writeOrganizationResult(ctx, http.StatusNoContent, nil, err)
}
