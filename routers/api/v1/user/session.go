package user

import (
	"net/http"

	auth_model "github.com/datvtph41107/bdspro/models/auth"
	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/services/context"
	"github.com/google/uuid"
)

// ListSessions returns only the sessions owned by the middleware-verified actor.
// No handler or service instance is needed for a model-owned read.
func ListSessions(ctx *context.APIContext) {
	actor, ok := ctx.VerifiedActor()
	if !ok {
		ctx.AuthenticationRequired()
		return
	}
	store, err := db.StoreFromContext(ctx)
	if err != nil {
		ctx.APIErrorAuto(err)
		return
	}
	sessions, err := auth_model.ListSessions(ctx, store, actor.AccountID)
	if err != nil {
		ctx.APIErrorAuto(err)
		return
	}
	if sessions == nil {
		sessions = []db.ListActiveSessionsRow{}
	}
	ctx.JSON(http.StatusOK, map[string]any{"sessions": sessions})
}

// RevokeOwnSession preserves the existing public error contract and delegates
// the locking, acting-session recheck, mutation and audit to the auth model.
func RevokeOwnSession(ctx *context.APIContext) {
	id, err := uuid.Parse(ctx.PathParam("id"))
	if err != nil {
		ctx.APIError(http.StatusBadRequest, "invalid_session_id", "invalid session id")
		return
	}
	actor, ok := ctx.VerifiedActor()
	if !ok {
		ctx.AuthenticationRequired()
		return
	}
	store, err := db.StoreFromContext(ctx)
	if err != nil {
		ctx.APIErrorAuto(err)
		return
	}
	if err := auth_model.RevokeOwnedSession(ctx, store, actor.AccountID, actor.SessionID, id); err != nil {
		ctx.APIErrorAuto(err)
		return
	}
	ctx.Status(http.StatusNoContent)
}

// RevokeAllSessions does not rely solely on HTTP admission: the model
// rechecks the acting session while holding the account lock in its transaction.
func RevokeAllSessions(ctx *context.APIContext) {
	actor, ok := ctx.VerifiedActor()
	if !ok {
		ctx.AuthenticationRequired()
		return
	}
	store, err := db.StoreFromContext(ctx)
	if err != nil {
		ctx.APIErrorAuto(err)
		return
	}
	if err := auth_model.RevokeAllSessions(ctx, store, actor.AccountID, actor.SessionID); err != nil {
		ctx.APIErrorAuto(err)
		return
	}
	ctx.Status(http.StatusNoContent)
}
