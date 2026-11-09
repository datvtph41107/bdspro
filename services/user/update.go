package user

import (
	"context"
	"strings"
	"unicode"
	"unicode/utf8"

	account_model "github.com/datvtph41107/bdspro/models/account"
	auth_model "github.com/datvtph41107/bdspro/models/auth"
	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/jackc/pgx/v5/pgtype"
)

type ProfilePatch struct {
	DisplayName *string
	FirstName   *string
	LastName    *string
}

// UpdateProfile locks the acting session and updates the user in ONE
// transaction; the response row is observable only after a successful commit.
func UpdateProfile(ctx context.Context, patch ProfilePatch) (db.GetAccountProfileRow, error) {
	actor, store, err := actingUser(ctx)
	if err != nil {
		return db.GetAccountProfileRow{}, err
	}
	accountID, actingSessionID := actor.AccountID, actor.SessionID
	if patch.DisplayName == nil && patch.FirstName == nil && patch.LastName == nil {
		return db.GetAccountProfileRow{}, account_model.ErrInvalidProfile
	}
	display, err := nameValue(patch.DisplayName)
	if err != nil {
		return db.GetAccountProfileRow{}, err
	}
	first, err := nameValue(patch.FirstName)
	if err != nil {
		return db.GetAccountProfileRow{}, err
	}
	last, err := nameValue(patch.LastName)
	if err != nil {
		return db.GetAccountProfileRow{}, err
	}
	return db.WithTxResult(ctx, store, func(q *db.Queries) (db.GetAccountProfileRow, error) {
		if err := auth_model.RequireActingSession(ctx, q, accountID, actingSessionID); err != nil {
			return db.GetAccountProfileRow{}, err
		}
		if err := account_model.UpdateProfile(ctx, q, db.UpdatePersonProfileParams{
			AccountID: accountID, DisplayName: display, FirstName: first, LastName: last,
		}); err != nil {
			return db.GetAccountProfileRow{}, err
		}
		// Read the response from the same transaction and publish it only
		// after the account update has successfully committed.
		return account_model.GetProfile(ctx, q, accountID)
	})
}

// nil preserves existing data; explicit empty strings clear a name.
func nameValue(p *string) (pgtype.Text, error) {
	if p == nil {
		return pgtype.Text{}, nil
	}
	value := strings.TrimSpace(*p)
	if utf8.RuneCountInString(value) > 120 || strings.IndexFunc(value, unicode.IsControl) >= 0 {
		return pgtype.Text{}, account_model.ErrInvalidProfile
	}
	return pgtype.Text{String: value, Valid: true}, nil
}
