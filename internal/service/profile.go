package service

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	db "github.com/datvtph41107/bdspro/internal/database"
	apperrors "github.com/datvtph41107/bdspro/internal/errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// ProfileService reads and changes data owned by the signed-in account.
// Account identity always comes from the verified access token.
type ProfileService struct {
	store db.Store
}

func NewProfileService(store db.Store) *ProfileService {
	return &ProfileService{store: store}
}

func (s *ProfileService) Show(ctx context.Context, accountID int64) (db.GetAccountProfileRow, error) {
	profile, err := s.store.GetAccountProfile(ctx, accountID)
	if errors.Is(err, pgx.ErrNoRows) {
		return db.GetAccountProfileRow{}, apperrors.AccountUnavailable
	}
	return profile, err
}

// Update changes only the fields explicitly present in PATCH. Each SQL
// COALESCE expression runs in one UPDATE, so independent concurrent field
// changes cannot overwrite each other through a read/modify/write race.
func (s *ProfileService) Update(ctx context.Context, accountID int64, displayName, firstName, lastName *string) (db.GetAccountProfileRow, error) {
	if displayName == nil && firstName == nil && lastName == nil {
		return db.GetAccountProfileRow{}, apperrors.InvalidProfile
	}
	display, err := profileField(displayName)
	if err != nil {
		return db.GetAccountProfileRow{}, err
	}
	first, err := profileField(firstName)
	if err != nil {
		return db.GetAccountProfileRow{}, err
	}
	last, err := profileField(lastName)
	if err != nil {
		return db.GetAccountProfileRow{}, err
	}
	n, err := s.store.UpdatePersonProfile(ctx, db.UpdatePersonProfileParams{
		AccountID: accountID, DisplayName: display, FirstName: first, LastName: last,
	})
	if err != nil {
		return db.GetAccountProfileRow{}, err
	}
	if n != 1 {
		return db.GetAccountProfileRow{}, apperrors.AccountUnavailable
	}
	return s.Show(ctx, accountID)
}

func profileField(value *string) (pgtype.Text, error) {
	if value == nil {
		return pgtype.Text{}, nil
	}
	trimmed := strings.TrimSpace(*value)
	if utf8.RuneCountInString(trimmed) > 120 {
		return pgtype.Text{}, apperrors.InvalidProfile
	}
	return pgtype.Text{String: trimmed, Valid: true}, nil
}

func (s *ProfileService) UpdateSettings(ctx context.Context, accountID int64, locale, timezone, currency string) (db.GetAccountProfileRow, error) {
	if (locale != "vi" && locale != "en") ||
		(timezone != "Asia/Ho_Chi_Minh" && timezone != "UTC") ||
		(currency != "VND" && currency != "USD") {
		return db.GetAccountProfileRow{}, apperrors.InvalidSettings
	}
	n, err := s.store.UpdateAccountSettings(ctx, db.UpdateAccountSettingsParams{
		AccountID: accountID, Locale: locale, Timezone: timezone, Currency: currency,
	})
	if err != nil {
		return db.GetAccountProfileRow{}, err
	}
	if n != 1 {
		return db.GetAccountProfileRow{}, apperrors.AccountUnavailable
	}
	return s.Show(ctx, accountID)
}
