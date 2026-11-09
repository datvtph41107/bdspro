package org

import (
	"net/http"

	organization_model "github.com/datvtph41107/bdspro/models/organization"
	appctx "github.com/datvtph41107/bdspro/services/context"
	"github.com/datvtph41107/bdspro/services/convert"
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
