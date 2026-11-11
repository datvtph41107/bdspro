// Package user owns user-scoped operations; it does not own HTTP transport.
package user

import (
	"context"

	account_model "github.com/datvtph41107/bdspro/models/account"
	auth_model "github.com/datvtph41107/bdspro/models/auth"
	db "github.com/datvtph41107/bdspro/models/db"
)

// actingUser validates the middleware-verified actor and the request DB
// capability before business validation. It does not open a transaction;
// callers select the executor needed by their read/write workflow.
func actingUser(ctx context.Context) (auth_model.Actor, error) {
	actor, ok := auth_model.ActorFromContext(ctx)
	if !ok {
		return auth_model.Actor{}, auth_model.ErrSessionUnavailable
	}
	if _, err := db.GetQueries(ctx); err != nil {
		return auth_model.Actor{}, err
	}
	return actor, nil
}

// GetAuthenticatedUser loads the currently signed-in account projection.
func GetAuthenticatedUser(ctx context.Context) (*account_model.Profile, error) {
	actor, err := actingUser(ctx)
	if err != nil {
		return nil, err
	}
	q, err := db.GetQueries(ctx)
	if err != nil {
		return nil, err
	}
	return account_model.GetProfile(ctx, q, actor.AccountID)
}
