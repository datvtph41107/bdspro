package apiv1

import (
	"errors"
	"net/http"

	"github.com/datvtph41107/bdspro/services/context"
	org_service "github.com/datvtph41107/bdspro/services/org"
)

func orgAssignment(ctx *context.APIContext) {
	if ctx.Doer == nil {
		ctx.AuthenticationRequired()
		return
	}
	orgID, err := ctx.PositivePathID("org")
	if err != nil {
		ctx.APIInvalid(err)
		return
	}
	org, err := org_service.GetOrganizationForAccount(ctx, ctx.Doer.AccountID, orgID)
	if err != nil {
		ctx.APIErrorAuto(err)
		return
	}
	ctx.Org = &context.APIOrganization{Organization: org}
}

var errUnpreparedOrganization = errors.New("org assignment not initialized")

func reqOrgOwner() func(*context.APIContext) {
	return func(ctx *context.APIContext) {
		if ctx.Org == nil || ctx.Org.Organization == nil {
			ctx.APIErrorInternal(errUnpreparedOrganization)
			return
		}
		if !ctx.Org.Organization.IsOwner() {
			ctx.APIError(http.StatusForbidden, "organization_owner_required", "organization owner permission required")
		}
	}
}
