package apiv1

import (
	"errors"
	"net/http"

	"github.com/datvtph41107/bdspro/services/context"
	org_service "github.com/datvtph41107/bdspro/services/org"
)

// orgAssignment is the only HTTP action that loads Organization into
// APIContext. It requires common.APIAuth to have run first.
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

// reqOrgOwner performs a fast precondition check. SQLC repeats this check
// under PostgreSQL row locks at each actual write, so this is not the sole
// authority under concurrent role changes.
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
