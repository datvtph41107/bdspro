package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	db "github.com/datvtph41107/bdspro/internal/database"
	apperrors "github.com/datvtph41107/bdspro/internal/errors"
)

type profileStoreFake struct {
	db.Store
	updates int
}

func (s *profileStoreFake) UpdatePersonProfile(context.Context, db.UpdatePersonProfileParams) (int64, error) {
	s.updates++
	return 1, nil
}

func (s *profileStoreFake) UpdateAccountSettings(context.Context, db.UpdateAccountSettingsParams) (int64, error) {
	s.updates++
	return 1, nil
}

func (s *profileStoreFake) GetAccountProfile(context.Context, int64) (db.GetAccountProfileRow, error) {
	return db.GetAccountProfileRow{AccountID: 7, Locale: "vi"}, nil
}

func TestProfileRejectsInvalidInputBeforeDatabase(t *testing.T) {
	store := &profileStoreFake{}
	profile := NewProfileService(store)
	oversized := strings.Repeat("a", 121)
	if _, err := profile.Update(context.Background(), 7, &oversized, nil, nil); !errors.Is(err, apperrors.InvalidProfile) {
		t.Fatalf("oversized profile must fail: %v", err)
	}
	if _, err := profile.Update(context.Background(), 7, nil, nil, nil); !errors.Is(err, apperrors.InvalidProfile) {
		t.Fatalf("empty patch must fail: %v", err)
	}
	if _, err := profile.UpdateSettings(context.Background(), 7, "unknown", "UTC", "USD"); !errors.Is(err, apperrors.InvalidSettings) {
		t.Fatalf("unsupported settings must fail: %v", err)
	}
	if store.updates != 0 {
		t.Fatalf("invalid values wrote to PostgreSQL: %d", store.updates)
	}
}
