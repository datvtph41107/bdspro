package apiv1

import (
	"errors"
	"strconv"

	db "github.com/datvtph41107/bdspro/models/db"
	organization_model "github.com/datvtph41107/bdspro/models/organization"
	"github.com/datvtph41107/bdspro/modules/web"
	appctx "github.com/datvtph41107/bdspro/services/context"
)

// orgAssignment loads the resource visible through the caller's active
// Account -> Person -> Organization membership. It is not write authority.
func orgAssignment() func(*appctx.APIContext) {
	return func(ctx *appctx.APIContext) {
		id, err := strconv.ParseInt(ctx.PathParam("id"), 10, 64)
		if err != nil || id <= 0 {
			ctx.APIErrorAuto(organization_model.ErrInvalidOrganization)
			return
		}
		store, err := db.StoreFromContext(ctx)
		if err != nil {
			ctx.APIErrorAuto(err)
			return
		}
		access, err := organization_model.GetForAccount(ctx, store, ctx.Doer.AccountID, id)
		if errors.Is(err, organization_model.ErrOrganizationNotFound) {
			// Preserve the established Team CREATE owner-denial response.
			ctx.APIErrorAuto(organization_model.ErrTeamOwnerRequired)
			return
		}
		if err != nil {
			ctx.APIErrorAuto(err)
			return
		}
		ctx.Org = &appctx.APIOrganization{
			Organization: access.OrganizationModel(),
			Access: &access,
		}
	}
}

// teamAssignment resolves a Team from its own route rather than treating
// Team ID as Organization ID. The SQLC read is account-access scoped.
func teamAssignment() func(*appctx.APIContext) {
	return func(ctx *appctx.APIContext) {
		id, err := strconv.ParseInt(ctx.PathParam("team_id"), 10, 64)
		if err != nil || id <= 0 {
			ctx.APIInvalid(err)
			return
		}
		store, err := db.StoreFromContext(ctx)
		if err != nil {
			ctx.APIErrorAuto(err)
			return
		}
		row, err := organization_model.GetTeam(ctx, store, ctx.Doer.AccountID, id)
		if err != nil {
			ctx.APIErrorAuto(err)
			return
		}
		access, err := organization_model.GetForAccount(ctx, store, ctx.Doer.AccountID, row.OrganizationID)
		if err != nil {
			ctx.APIErrorAuto(err)
			return
		}
		ctx.Org = &appctx.APIOrganization{
			Organization: access.OrganizationModel(),
			Team: organization_model.TeamFromDB(row),
			Access: &access,
		}
	}
}

// reqToken verifies the result of upstream authentication/session middleware.
// JWT verification is not duplicated here.
func reqToken() func(*appctx.APIContext) {
	return func(ctx *appctx.APIContext) {
		if ctx.IsSigned && ctx.Doer != nil && ctx.Doer.AccountID > 0 {
			return
		}
		appctx.AuthenticationRequired(ctx.Context)
	}
}

func reqOrgMembership() func(*appctx.APIContext) {
	return func(ctx *appctx.APIContext) {
		if ctx.Org == nil || ctx.Org.Organization == nil || ctx.Org.Access == nil {
			ctx.APIErrorAuto(organization_model.ErrOrganizationNotFound)
		}
	}
}

// Only a preflight permission check. SQLC owns commit-order authority.
func reqOrgOwnership() func(*appctx.APIContext) {
	return func(ctx *appctx.APIContext) {
		if ctx.Org == nil || ctx.Org.Organization == nil || ctx.Org.Access == nil {
			ctx.APIErrorAuto(organization_model.ErrOrganizationNotFound)
			return
		}
		if ctx.Org.Access.Role != organization_model.RoleOwner {
			ctx.APIErrorAuto(organization_model.ErrTeamOwnerRequired)
		}
	}
}

// bind validates once and saves a typed, per-request form for the action.
func bind[T any](_ T) func(*appctx.APIContext) {
	return func(ctx *appctx.APIContext) {
		var form T
		if err := ctx.ShouldBindJSON(&form); err != nil {
			appctx.Validation(ctx.Context, err, &form)
			return
		}
		web.SetForm(ctx, &form)
	}
}
