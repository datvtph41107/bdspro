// Package user owns user-scoped operations; it does not own HTTP transport.
package user

import (
	"context"

	account_model "github.com/datvtph41107/bdspro/models/account"
	auth_model "github.com/datvtph41107/bdspro/models/auth"
	db "github.com/datvtph41107/bdspro/models/db"
)

// actingUser resolves the middleware-verified actor; it does not open or
// select a SQL executor. The caller's read/write workflow owns DB access.
func actingUser(ctx context.Context) (auth_model.Actor, error) {
	actor, ok := auth_model.ActorFromContext(ctx)
	if !ok {
		return auth_model.Actor{}, auth_model.ErrSessionUnavailable
	}
	return actor, nil
}

// GetAuthenticatedUser loads the currently signed-in account projection.
func GetAuthenticatedUser(ctx context.Context) (*account_model.Profile, error) {
	actor, err := actingUser(ctx)
	if err != nil {
		return nil, err
	}
	store, err := db.StoreFromContext(ctx)
	if err != nil {
		return nil, err
	}
	return account_model.GetProfile(ctx, store, actor.AccountID)
}
