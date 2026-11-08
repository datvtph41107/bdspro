// Package profile owns account-scoped personal details and preferences.
// It does not authenticate callers: the HTTP boundary supplies the verified AccountID.
package profile

import (
	"context"
	"strings"
	"unicode"
	"unicode/utf8"

	account_model "github.com/datvtph41107/bdspro/models/account"
	auth_model "github.com/datvtph41107/bdspro/models/auth"
	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type Service struct{ store db.Store }

func NewService(store db.Store) *Service { return &Service{store: store} }

func (s *Service) Show(ctx context.Context, accountID int64) (db.GetAccountProfileRow, error) {
	return account_model.GetProfile(ctx, s.store, accountID)
}

type ProfilePatch struct {
	DisplayName *string
	FirstName   *string
	LastName    *string
}

func (s *Service) Update(ctx context.Context, accountID int64, actingSessionID uuid.UUID, patch ProfilePatch) (db.GetAccountProfileRow, error) {
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
	return db.WithTxResult(ctx, s.store, func(q *db.Queries) (db.GetAccountProfileRow, error) {
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

type SettingsPatch struct {
	Locale   *string
	Timezone *string
	Currency *string
}

func (s *Service) UpdateSettings(ctx context.Context, accountID int64, actingSessionID uuid.UUID, patch SettingsPatch) (db.GetAccountProfileRow, error) {
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
	return db.WithTxResult(ctx, s.store, func(q *db.Queries) (db.GetAccountProfileRow, error) {
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
