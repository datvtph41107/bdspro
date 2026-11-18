package auth

import (
	"net/http"

	authservice "github.com/datvtph41107/bdspro/services/auth"
	appctx "github.com/datvtph41107/bdspro/services/context"
)

// Logout revokes the caller's own verified, live session.
// Other sessions remain active; request bodies cannot select a target.
func Logout(ctx *appctx.APIContext) {
	if ctx.Doer == nil || !ctx.IsSigned {
		ctx.AuthenticationRequired()
		return
	}

	revoked, err := authservice.RevokeSession(ctx, ctx.Doer, ctx.Doer.SessionID)
	if err != nil {
		ctx.APIErrorAuto(err)
		return
	}
	if !revoked {
		ctx.AuthenticationRequired()
		return
	}
	ctx.Status(http.StatusNoContent)
}
