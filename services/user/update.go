package user

import (
	"context"

	account_model "github.com/datvtph41107/bdspro/models/account"
	auth_model "github.com/datvtph41107/bdspro/models/auth"
	db "github.com/datvtph41107/bdspro/models/db"
)

// ProfilePatch contains optional user profile fields accepted by the service.
// Model validation and PostgreSQL-specific conversion belong to models/account.
type ProfilePatch struct {
	DisplayName *string
	FirstName   *string
	LastName    *string
}

// UpdateProfile coordinates a live-session lock, the model write, and a
// same-transaction read. No driver value or generated SQLC param escapes
// models/account; no result is returned when the commit fails.
func UpdateProfile(ctx context.Context, patch ProfilePatch) (db.GetAccountProfileRow, error) {
	actor, store, err := actingUser(ctx)
	if err != nil {
		return db.GetAccountProfileRow{}, err
	}
	opts := account_model.ProfileUpdateOptions{
		DisplayName: patch.DisplayName,
		FirstName: patch.FirstName,
		LastName: patch.LastName,
	}
	if err := opts.Validate(); err != nil {
		return db.GetAccountProfileRow{}, err
	}
	return db.WithTxResult(ctx, store, func(q *db.Queries) (db.GetAccountProfileRow, error) {
		if err := auth_model.RequireActingSession(ctx, q, actor.AccountID, actor.SessionID); err != nil {
			return db.GetAccountProfileRow{}, err
		}
		if err := account_model.UpdateProfile(ctx, q, actor.AccountID, opts); err != nil {
			return db.GetAccountProfileRow{}, err
		}
		return account_model.GetProfile(ctx, q, actor.AccountID)
	})
}
