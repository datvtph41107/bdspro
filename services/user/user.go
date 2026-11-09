// Package user owns user-scoped operations; it does not own HTTP transport.
package user

import (
	"context"

	account_model "github.com/datvtph41107/bdspro/models/account"
	auth_model "github.com/datvtph41107/bdspro/models/auth"
	db "github.com/datvtph41107/bdspro/models/db"
)

// actingUser resolves the middleware-verified actor and request-scoped
// database capability. Neither JWT claims nor HTTP handlers grant writes.
func actingUser(ctx context.Context) (auth_model.Actor, db.Store, error) {
	actor, ok := auth_model.ActorFromContext(ctx)
	if !ok {
		return auth_model.Actor{}, nil, auth_model.ErrSessionUnavailable
	}
	store, err := db.StoreFromContext(ctx)
	if err != nil {
		return auth_model.Actor{}, nil, err
	}
	return actor, store, nil
}

// GetAuthenticatedUser loads the currently signed-in account projection.
func GetAuthenticatedUser(ctx context.Context) (*account_model.Profile, error) {
	actor, store, err := actingUser(ctx)
	if err != nil {
		return nil, err
	}
	return account_model.GetProfile(ctx, store, actor.AccountID)
}
