package org

import (
	"net/http"
	"strings"

	organization_model "github.com/datvtph41107/bdspro/models/organization"
	api "github.com/datvtph41107/bdspro/modules/structs"
	"github.com/datvtph41107/bdspro/modules/web"
	appctx "github.com/datvtph41107/bdspro/services/context"
	"github.com/datvtph41107/bdspro/services/convert"
	org_service "github.com/datvtph41107/bdspro/services/org"
)

// List reads only organizations visible to the verified current Account.
func List(ctx *appctx.APIContext) {
	orgs, err := organization_model.ListForDoer(ctx)
	if err != nil {
		ctx.APIErrorAuto(err)
		return
	}
	ctx.JSON(http.StatusOK, convert.ToOrganizationAccessList(orgs))
}

// Get uses the Organization already loaded by orgAssignment.
// It never repeats the same membership query in the HTTP action.
func Get(ctx *appctx.APIContext) {
	if ctx.Org == nil || ctx.Org.Access == nil {
		ctx.APIErrorAuto(organization_model.ErrOrganizationNotFound)
		return
	}
	ctx.JSON(http.StatusOK, convert.ToOrganizationAccess(*ctx.Org.Access))
}

// Create constructs the Organization model from a typed form. All account,
// session and owner writes remain transaction-scoped in org_service.
func Create(ctx *appctx.APIContext) {
	form := web.GetForm[*api.CreateOrgOption](ctx)
	if strings.TrimSpace(form.Name) == "" {
		ctx.FieldError("name", "required", "name is required")
		return
	}
	org := &organization_model.Organization{Name: form.Name}
	if err := org_service.NewOrganization(ctx, org); err != nil {
		ctx.APIErrorAuto(err)
		return
	}
	ctx.JSON(http.StatusCreated, convert.ToOrganizationModel(org))
}

// Edit uses the assigned Organization ID, never an ID supplied by the body.
// Its write is authorized again inside a SQLC transaction.
func Edit(ctx *appctx.APIContext) {
	form := web.GetForm[*api.EditOrgOption](ctx)
	if form.Name == nil || strings.TrimSpace(*form.Name) == "" {
		ctx.FieldError("name", "required", "name is required")
		return
	}
	org := &organization_model.Organization{
		ID:   ctx.Org.Organization.ID,
		Name: *form.Name,
	}
	if err := org_service.EditOrganization(ctx, org); err != nil {
		ctx.APIErrorAuto(err)
		return
	}
	ctx.JSON(http.StatusOK, convert.ToOrganizationModel(org))
}
