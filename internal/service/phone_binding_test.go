package service

import (
	"context"
	"errors"
	"testing"

	"github.com/datvtph41107/bdspro/internal/config"
	db "github.com/datvtph41107/bdspro/internal/database"
	apperrors "github.com/datvtph41107/bdspro/internal/errors"
	"github.com/jackc/pgx/v5"
)

type phoneBindingStore struct {
	db.Store
	linkedID      int64
	linkedSub     string
	accountID     int64
	existingPhone int64
	alreadyBound  bool
	issued        int
	lastCode      string
}

func (s *phoneBindingStore) GetGoogleAccountID(_ context.Context, sub string) (int64, error) {
	if sub != s.linkedSub {
		return 0, pgx.ErrNoRows
	}
	return s.linkedID, nil
}

func (s *phoneBindingStore) GetAccountByID(_ context.Context, id int64) (db.Account, error) {
	if id != s.accountID {
		return db.Account{}, pgx.ErrNoRows
	}
	return db.Account{ID: id, IsEmailVerified: true}, nil
}

func (s *phoneBindingStore) GetPhoneForAccount(_ context.Context, _ int64) (string, error) {
	if s.alreadyBound {
		return "+84988887777", nil
	}
	return "", pgx.ErrNoRows
}

func (s *phoneBindingStore) AccountIDForPhone(_ context.Context, _ string) (int64, error) {
	if s.existingPhone != 0 {
		return s.existingPhone, nil
	}
	return 0, pgx.ErrNoRows
}

func (s *phoneBindingStore) StartChallenge(_ context.Context, _, _ string, _ int64, _ []byte, code string) error {
	s.issued++
	s.lastCode = code
	return nil
}

func TestGoogleSubjectMustOwnPhoneBindingAccount(t *testing.T) {
	ctx := context.Background()
	store := &phoneBindingStore{accountID: 3, linkedID: 3, linkedSub: "google-sub-3"}
	phone := PhoneOTP{Store: store, Config: config.Config{
		Environment: "development", TokenSymmetricKey: "01234567890123456789012345678901",
	}}
	for _, tc := range []struct {
		name string
		id   int64
		sub  string
	}{
		{"unlinked subject", 3, "attacker"},
		{"wrong account", 4, "google-sub-3"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := phone.StartGoogleBind(ctx, tc.id, tc.sub, "+84912345678"); !errors.Is(err, apperrors.InvalidCredentials) {
				t.Fatalf("expected identity rejection, got %v", err)
			}
		})
	}
	if store.issued != 0 {
		t.Fatal("unverified Google identity issued an OTP")
	}
	if err := phone.StartGoogleBind(ctx, 3, "google-sub-3", "+84912345678"); err != nil {
		t.Fatal(err)
	}
	if store.issued != 1 || len(store.lastCode) != 6 {
		t.Fatalf("expected exactly one local OTP, got %d", store.issued)
	}
	store.existingPhone = 11
	if err := phone.StartGoogleBind(ctx, 3, "google-sub-3", "+84912345678"); !errors.Is(err, apperrors.PhoneUnavailable) {
		t.Fatalf("already bound number should be unavailable: %v", err)
	}
	if store.issued != 1 {
		t.Fatal("already bound number caused OTP delivery")
	}
	store.existingPhone = 0
	store.alreadyBound = true
	if err := phone.StartGoogleBind(ctx, 3, "google-sub-3", "+84912345679"); !errors.Is(err, apperrors.PhoneUnavailable) {
		t.Fatalf("already-bound account issued OTP: %v", err)
	}
	if store.issued != 1 {
		t.Fatal("already-bound account caused OTP delivery")
	}
}
