package user

import (
	"context"

	account_model "github.com/datvtph41107/bdspro/models/account"
	auth_model "github.com/datvtph41107/bdspro/models/auth"
	db "github.com/datvtph41107/bdspro/models/db"
)

// SettingsPatch describes a partial user settings update.
type SettingsPatch struct {
	Locale   *string
	Timezone *string
	Currency *string
}

// UpdateUserSettings coordinates validation, current-session authority, a
// model-owned SQLC update, and the projection read within one transaction.
func UpdateUserSettings(ctx context.Context, patch SettingsPatch) (*account_model.Profile, error) {
	actor, store, err := actingUser(ctx)
	if err != nil {
		return nil, err
	}
	opts := account_model.SettingsUpdateOptions{
		Locale:   patch.Locale,
		Timezone: patch.Timezone,
		Currency: patch.Currency,
	}
	if err := opts.Validate(); err != nil {
		return nil, err
	}
	return db.WithTxResult(ctx, store, func(q *db.Queries) (*account_model.Profile, error) {
		if err := auth_model.RequireActingSession(ctx, q, actor.AccountID, actor.SessionID); err != nil {
			return nil, err
		}
		if err := account_model.UpdateSettings(ctx, q, actor.AccountID, opts); err != nil {
			return nil, err
		}
		return account_model.GetProfile(ctx, q, actor.AccountID)
	})
}
