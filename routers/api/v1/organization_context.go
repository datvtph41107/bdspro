package apiv1

import (
	"errors"

	db "github.com/datvtph41107/bdspro/models/db"
	organization_model "github.com/datvtph41107/bdspro/models/organization"
	"github.com/datvtph41107/bdspro/modules/web"
	webmiddleware "github.com/datvtph41107/bdspro/modules/web/middleware"
	appctx "github.com/datvtph41107/bdspro/services/context"
)

// orgAssignment loads the resource visible through the caller's active
// Account -> Person -> Organization membership. It is not write authority.
func orgAssignment(denied error) func(*appctx.APIContext) {
	return func(ctx *appctx.APIContext) {
		actor, ok := ctx.VerifiedActor()
		if !ok {
			ctx.AuthenticationRequired()
			return
		}
		id, err := ctx.PositivePathID("id")
		if err != nil {
			ctx.APIErrorAuto(organization_model.ErrInvalidOrganization)
			return
		}
		store, err := db.StoreFromContext(ctx)
		if err != nil {
			ctx.APIErrorAuto(err)
			return
		}
		access, err := organization_model.GetForAccount(ctx, store, actor.AccountID, id)
		if errors.Is(err, organization_model.ErrOrganizationNotFound) {
			if denied != nil {
				// Preserve the owner write endpoint's existing public error.
				ctx.APIErrorAuto(denied)
			} else {
				ctx.APIErrorAuto(organization_model.ErrOrganizationNotFound)
			}
			return
		}
		if err != nil {
			ctx.APIErrorAuto(err)
			return
		}
		ctx.Org = &appctx.APIOrganization{
			Organization: access.OrganizationModel(),
			Access:       &access,
		}
	}
}

// teamAssignment resolves a Team from its own route rather than treating
// Team ID as Organization ID. The SQLC read is account-access scoped.
func teamAssignment() func(*appctx.APIContext) {
	return func(ctx *appctx.APIContext) {
		actor, ok := ctx.VerifiedActor()
		if !ok {
			ctx.AuthenticationRequired()
			return
		}
		id, err := ctx.PositivePathID("team_id")
		if err != nil {
			ctx.APIInvalid(err)
			return
		}
		store, err := db.StoreFromContext(ctx)
		if err != nil {
			ctx.APIErrorAuto(err)
			return
		}
		row, err := organization_model.GetTeam(ctx, store, actor.AccountID, id)
		if err != nil {
			ctx.APIErrorAuto(err)
			return
		}
		access, err := organization_model.GetForAccount(ctx, store, actor.AccountID, row.OrganizationID)
		if err != nil {
			ctx.APIErrorAuto(err)
			return
		}
		ctx.Org = &appctx.APIOrganization{
			Organization: access.OrganizationModel(),
			Team:         organization_model.TeamFromDB(row),
			Access:       &access,
		}
	}
}

// reqToken verifies the result of upstream authentication/session middleware.
// JWT verification is not duplicated here.
func reqToken() func(*appctx.APIContext) {
	return func(ctx *appctx.APIContext) {
		if _, ok := ctx.VerifiedActor(); ok {
			return
		}
		ctx.AuthenticationRequired()
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
func reqOrgOwnership(denied ...error) func(*appctx.APIContext) {
	ownerError := error(organization_model.ErrTeamOwnerRequired)
	if len(denied) > 0 && denied[0] != nil {
		ownerError = denied[0]
	}
	return func(ctx *appctx.APIContext) {
		if ctx.Org == nil || ctx.Org.Organization == nil || ctx.Org.Access == nil {
			ctx.APIErrorAuto(organization_model.ErrOrganizationNotFound)
			return
		}
		if ctx.Org.Access.Role != organization_model.RoleOwner {
			ctx.APIErrorAuto(ownerError)
		}
	}
}

// bind validates once and saves a typed, per-request form for the action.
func bind[T any](_ T) func(*appctx.APIContext) {
	return func(ctx *appctx.APIContext) {
		form, err := webmiddleware.BindJSON[T](ctx)
		if err != nil {
			ctx.Validation(err, form)
			return
		}
		web.SetForm(ctx, form)
	}
}
