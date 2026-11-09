package user

import (
	"context"

	account_model "github.com/datvtph41107/bdspro/models/account"
	auth_model "github.com/datvtph41107/bdspro/models/auth"
	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/jackc/pgx/v5/pgtype"
)

type SettingsPatch struct {
	Locale   *string
	Timezone *string
	Currency *string
}

// UpdateUserSettings keeps partial-field semantics and the session lock
// within the same SQLC transaction as the settings change.
func UpdateUserSettings(ctx context.Context, patch SettingsPatch) (db.GetAccountProfileRow, error) {
	actor, store, err := actingUser(ctx)
	if err != nil {
		return db.GetAccountProfileRow{}, err
	}
	accountID, actingSessionID := actor.AccountID, actor.SessionID
	if patch.Locale == nil && patch.Timezone == nil && patch.Currency == nil {
		return db.GetAccountProfileRow{}, account_model.ErrInvalidSettings
	}
	locale, ok := option(patch.Locale, "vi", "en")
	if !ok {
		return db.GetAccountProfileRow{}, account_model.ErrInvalidSettings
	}
	tz, ok := option(patch.Timezone, "Asia/Ho_Chi_Minh", "UTC")
	if !ok {
		return db.GetAccountProfileRow{}, account_model.ErrInvalidSettings
	}
	currency, ok := option(patch.Currency, "VND", "USD")
	if !ok {
		return db.GetAccountProfileRow{}, account_model.ErrInvalidSettings
	}
	return db.WithTxResult(ctx, store, func(q *db.Queries) (db.GetAccountProfileRow, error) {
		if err := auth_model.RequireActingSession(ctx, q, accountID, actingSessionID); err != nil {
			return db.GetAccountProfileRow{}, err
		}
		if err := account_model.UpdateSettings(ctx, q, db.UpdateAccountSettingsParams{
			AccountID: accountID, Locale: locale, Timezone: tz, Currency: currency,
		}); err != nil {
			return db.GetAccountProfileRow{}, err
		}
		return account_model.GetProfile(ctx, q, accountID)
	})
}

func option(p *string, allowed ...string) (pgtype.Text, bool) {
	if p == nil {
		return pgtype.Text{}, true
	}
	for _, value := range allowed {
		if *p == value {
			return pgtype.Text{String: value, Valid: true}, true
		}
	}
	return pgtype.Text{}, false
}
