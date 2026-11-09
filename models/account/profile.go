package account

import (
	"context"
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"

	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// ProfileQueries is the SQLC capability for account profile persistence.
// The caller's verified account ID must never be replaced with client input.
type ProfileQueries interface {
	GetAccountProfile(context.Context, int64) (db.GetAccountProfileRow, error)
	UpdatePersonProfile(context.Context, db.UpdatePersonProfileParams) (int64, error)
	UpdateAccountSettings(context.Context, db.UpdateAccountSettingsParams) (int64, error)
}

// ProfileUpdateOptions is the model-level partial update contract. Nil leaves a
// field unchanged; an explicit empty string clears a name. PostgreSQL-specific
// pgtype values and generated query arguments are private to this package.
type ProfileUpdateOptions struct {
	DisplayName *string
	FirstName   *string
	LastName    *string
}

func (opts ProfileUpdateOptions) Validate() error {
	if opts.DisplayName == nil && opts.FirstName == nil && opts.LastName == nil {
		return ErrInvalidProfile
	}
	for _, field := range []*string{opts.DisplayName, opts.FirstName, opts.LastName} {
		if field == nil {
			continue
		}
		value := strings.TrimSpace(*field)
		if utf8.RuneCountInString(value) > 120 || strings.IndexFunc(value, unicode.IsControl) >= 0 {
			return ErrInvalidProfile
		}
	}
	return nil
}

// SettingsUpdateOptions is the model-level partial update contract.
type SettingsUpdateOptions struct {
	Locale   *string
	Timezone *string
	Currency *string
}

func (opts SettingsUpdateOptions) Validate() error {
	if opts.Locale == nil && opts.Timezone == nil && opts.Currency == nil {
		return ErrInvalidSettings
	}
	if !validSetting(opts.Locale, "vi", "en") ||
		!validSetting(opts.Timezone, "Asia/Ho_Chi_Minh", "UTC") ||
		!validSetting(opts.Currency, "VND", "USD") {
		return ErrInvalidSettings
	}
	return nil
}

func validSetting(value *string, allowed ...string) bool {
	if value == nil {
		return true
	}
	for _, candidate := range allowed {
		if *value == candidate {
			return true
		}
	}
	return false
}

func profileText(value *string) pgtype.Text {
	if value == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: strings.TrimSpace(*value), Valid: true}
}

func settingText(value *string) pgtype.Text {
	if value == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *value, Valid: true}
}

func GetProfile(ctx context.Context, q ProfileQueries, accountID int64) (db.GetAccountProfileRow, error) {
	if accountID <= 0 {
		return db.GetAccountProfileRow{}, ErrAccountUnavailable
	}
	profile, err := q.GetAccountProfile(ctx, accountID)
	if errors.Is(err, pgx.ErrNoRows) {
		return db.GetAccountProfileRow{}, ErrAccountUnavailable
	}
	return profile, err
}

// UpdateProfile owns SQLC-specific field mapping and row-count validation.
// The service owns the acting-session lock and transaction lifecycle.
func UpdateProfile(ctx context.Context, q ProfileQueries, accountID int64, opts ProfileUpdateOptions) error {
	if accountID <= 0 {
		return ErrAccountUnavailable
	}
	if err := opts.Validate(); err != nil {
		return err
	}
	affected, err := q.UpdatePersonProfile(ctx, db.UpdatePersonProfileParams{
		AccountID: accountID,
		DisplayName: profileText(opts.DisplayName),
		FirstName: profileText(opts.FirstName),
		LastName: profileText(opts.LastName),
	})
	if err != nil {
		return err
	}
	if affected != 1 {
		return ErrAccountUnavailable
	}
	return nil
}

// UpdateSettings owns allowed persisted values and SQLC-specific conversion.
func UpdateSettings(ctx context.Context, q ProfileQueries, accountID int64, opts SettingsUpdateOptions) error {
	if accountID <= 0 {
		return ErrAccountUnavailable
	}
	if err := opts.Validate(); err != nil {
		return err
	}
	affected, err := q.UpdateAccountSettings(ctx, db.UpdateAccountSettingsParams{
		AccountID: accountID,
		Locale: settingText(opts.Locale),
		Timezone: settingText(opts.Timezone),
		Currency: settingText(opts.Currency),
	})
	if err != nil {
		return err
	}
	if affected != 1 {
		return ErrAccountUnavailable
	}
	return nil
}
