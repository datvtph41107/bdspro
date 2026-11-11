package context

import (
	"errors"
	"net/http"
	"time"

	auth_model "github.com/datvtph41107/bdspro/models/auth"
	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/modules/log"
)

// RequireSession is the typed live-session admission step used after
// Authenticate. It reads the context-bound store, never a per-route Store
// argument. Commit-time authority still belongs to the model transaction.
func RequireSession() func(*APIContext) {
	return func(ctx *APIContext) {
		payload, ok := ctx.SignedPrincipal()
		if !ok {
			ctx.AuthenticationRequired()
			return
		}
		_, err := db.GetQueries(ctx)
		if err != nil {
			log.Ctx(ctx).Error().Str("dependency", "postgres").Msg("auth.identity_store_unavailable")
			ctx.APIError(http.StatusServiceUnavailable, "identity_store_unavailable", "identity store unavailable")
			return
		}
		err = auth_model.RequireActiveSession(ctx,
			payload.AccountID, payload.SessionID, time.Now())
		switch {
		case errors.Is(err, auth_model.ErrSessionUnavailable):
			ctx.Header("WWW-Authenticate", `Bearer error="invalid_token"`)
			ctx.APIError(http.StatusUnauthorized, "invalid_access_token", "invalid access token")
			return
		case err != nil:
			log.Ctx(ctx).Error().Str("dependency", "postgres").Msg("auth.identity_store_unavailable")
			ctx.APIError(http.StatusServiceUnavailable, "identity_store_unavailable", "identity store unavailable")
			return
		}
		ctx.PublishVerifiedActor(auth_model.Actor{
			AccountID: payload.AccountID,
			SessionID: payload.SessionID,
		})
	}
}
