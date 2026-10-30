package service

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	db "github.com/datvtph41107/bdspro/internal/database"
	apperrors "github.com/datvtph41107/bdspro/internal/errors"
	"github.com/jackc/pgx/v5"
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

func (s *ProfileService) Update(ctx context.Context, accountID int64, displayName, firstName, lastName string) (db.GetAccountProfileRow, error) {
	displayName = strings.TrimSpace(displayName)
	firstName = strings.TrimSpace(firstName)
	lastName = strings.TrimSpace(lastName)
	for _, value := range []string{displayName, firstName, lastName} {
		if utf8.RuneCountInString(value) > 120 {
			return db.GetAccountProfileRow{}, apperrors.InvalidProfile
		}
	}
	n, err := s.store.UpdatePersonProfile(ctx, db.UpdatePersonProfileParams{
		AccountID: accountID, DisplayName: displayName, FirstName: firstName, LastName: lastName,
	})
	if err != nil {
		return db.GetAccountProfileRow{}, err
	}
	if n != 1 {
		return db.GetAccountProfileRow{}, apperrors.AccountUnavailable
	}
	return s.Show(ctx, accountID)
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
