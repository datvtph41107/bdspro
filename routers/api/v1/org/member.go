package org

import (
	"net/http"

	organization_model "github.com/datvtph41107/bdspro/models/organization"
	"github.com/datvtph41107/bdspro/modules/structs"
	appctx "github.com/datvtph41107/bdspro/services/context"
	"github.com/datvtph41107/bdspro/services/convert"
	org_service "github.com/datvtph41107/bdspro/services/org"
)

// Members reads through the verified Account and account-scoped SQLC model.
func Members(ctx *appctx.APIContext) {
	id, ok := organizationID(ctx)
	if !ok {
		return
	}
	members, err := organization_model.GetOrgMembersForDoer(ctx, id)
	if err != nil {
		ctx.APIErrorAuto(err)
		return
	}
	ctx.JSON(http.StatusOK, convert.ToActiveOrganizationMembers(members))
}

func AddMember(ctx *appctx.APIContext) {
	id, ok := organizationID(ctx)
	if !ok {
		return
	}
	var req structs.AddOrgUserOption
	if err := ctx.BindJSON(&req); err != nil {
		ctx.Validation(err, &req)
		return
	}
	member, err := org_service.AddMember(ctx, id, req.PersonID)
	if err != nil {
		ctx.APIErrorAuto(err)
		return
	}
	ctx.JSON(http.StatusCreated, convert.ToOrganizationMember(member))
}

// RemoveMember ends an ordinary membership and its Team grants atomically.
// SQLC prevents removing an Organization owner through this action.
func RemoveMember(ctx *appctx.APIContext) {
	id, ok := organizationID(ctx)
	if !ok {
		return
	}
	personID, ok := positivePathID(ctx, "person_id")
	if !ok {
		return
	}
	if err := org_service.RemoveMember(ctx, id, personID); err != nil {
		ctx.APIErrorAuto(err)
		return
	}
	ctx.Status(http.StatusNoContent)
}

func PromoteOwner(ctx *appctx.APIContext) {
	id, ok := organizationID(ctx)
	if !ok {
		return
	}
	personID, ok := positivePathID(ctx, "person_id")
	if !ok {
		return
	}
	member, err := org_service.PromoteOwner(ctx, id, personID)
	if err != nil {
		ctx.APIErrorAuto(err)
		return
	}
	ctx.JSON(http.StatusOK, convert.ToOrganizationMember(member))
}

func DemoteOwner(ctx *appctx.APIContext) {
	id, ok := organizationID(ctx)
	if !ok {
		return
	}
	personID, ok := positivePathID(ctx, "person_id")
	if !ok {
		return
	}
	if err := org_service.DemoteOwner(ctx, id, personID); err != nil {
		ctx.APIErrorAuto(err)
		return
	}
	ctx.Status(http.StatusNoContent)
}

// TransferOwner is one transaction, never two independent role changes.
func TransferOwner(ctx *appctx.APIContext) {
	id, ok := organizationID(ctx)
	if !ok {
		return
	}
	var req structs.TransferOrganizationOwnerOption
	if err := ctx.BindJSON(&req); err != nil {
		ctx.Validation(err, &req)
		return
	}
	member, err := org_service.TransferOwner(ctx, id, req.PersonID)
	if err != nil {
		ctx.APIErrorAuto(err)
		return
	}
	ctx.JSON(http.StatusOK, convert.ToOrganizationMember(member))
}

func organizationID(ctx *appctx.APIContext) (int64, bool) {
	id, err := ctx.PositivePathID("id")
	if err != nil {
		ctx.APIErrorAuto(organization_model.ErrInvalidOrganization)
		return 0, false
	}
	return id, true
}
