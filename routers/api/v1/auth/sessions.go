package auth

import (
	"net/http"

	authservice "github.com/datvtph41107/bdspro/services/auth"
	appctx "github.com/datvtph41107/bdspro/services/context"
)

// ListSessions serves the authenticated account only; there is no
// accountID parameter and no ability to enumerate another user's sessions.
func ListSessions(ctx *appctx.APIContext) {
	if ctx.Doer == nil || !ctx.IsSigned {
		ctx.AuthenticationRequired()
		return
	}
	sessions, err := authservice.ListSessions(ctx, ctx.Doer)
	if err != nil {
		ctx.APIErrorAuto(err)
		return
	}
	ctx.Header("Cache-Control", "no-store")
	ctx.JSON(http.StatusOK, map[string]any{"items": sessions})
}
