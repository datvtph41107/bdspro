package auth

import (
	"context"
	"errors"
	auth_model "github.com/datvtph41107/bdspro/models/auth"
	"testing"

	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/modules/setting"
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

func (s *phoneBindingStore) GetOAuth2AccountID(_ context.Context, arg db.GetOAuth2AccountIDParams) (int64, error) {
	if arg.Issuer != "https://issuer.example.test" || arg.Subject != s.linkedSub {
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

func (s *phoneBindingStore) GetPhoneAccountIDQuery(_ context.Context, _ string) (int64, error) {
	if s.existingPhone != 0 {
		return s.existingPhone, nil
	}
	return 0, pgx.ErrNoRows
}

func (s *phoneBindingStore) UpsertPhoneChallengeQuery(_ context.Context, arg db.UpsertPhoneChallengeQueryParams) (int64, error) {
	s.issued++
	s.lastCode = arg.DevCode
	return 1, nil
}

func TestOAuth2CredentialMustOwnPhoneBindingAccount(t *testing.T) {
	ctx := context.Background()
	store := &phoneBindingStore{accountID: 3, linkedID: 3, linkedSub: "subject-3"}
	phone := NewAuthService(store, nil, setting.Config{
		Environment: "development", TokenSymmetricKey: "01234567890123456789012345678901",
		OIDC: setting.OAuth2Provider{Issuer: "https://issuer.example.test"},
	}, nil)
	for _, tc := range []struct {
		name string
		id   int64
		sub  string
	}{
		{"unlinked subject", 3, "attacker"},
		{"wrong account", 4, "subject-3"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := phone.StartOAuth2PhoneBind(ctx, tc.id, auth_model.OIDCClaims{Issuer: "https://issuer.example.test", Subject: tc.sub}, "+84912345678"); !errors.Is(err, auth_model.ErrInvalidCredentials) {
				t.Fatalf("expected identity rejection, got %v", err)
			}
		})
	}
	if store.issued != 0 {
		t.Fatal("unverified OAuth2 credential issued an OTP")
	}
	if err := phone.StartOAuth2PhoneBind(ctx, 3, auth_model.OIDCClaims{Issuer: "https://issuer.example.test", Subject: "subject-3"}, "+84912345678"); err != nil {
		t.Fatal(err)
	}
	if store.issued != 1 || len(store.lastCode) != 6 {
		t.Fatalf("expected exactly one local OTP, got %d", store.issued)
	}
	store.existingPhone = 11
	if err := phone.StartOAuth2PhoneBind(ctx, 3, auth_model.OIDCClaims{Issuer: "https://issuer.example.test", Subject: "subject-3"}, "+84912345678"); !errors.Is(err, auth_model.ErrPhoneUnavailable) {
		t.Fatalf("already bound number should be unavailable: %v", err)
	}
	if store.issued != 1 {
		t.Fatal("already bound number caused OTP delivery")
	}
	store.existingPhone = 0
	store.alreadyBound = true
	if err := phone.StartOAuth2PhoneBind(ctx, 3, auth_model.OIDCClaims{Issuer: "https://issuer.example.test", Subject: "subject-3"}, "+84912345679"); !errors.Is(err, auth_model.ErrPhoneUnavailable) {
		t.Fatalf("already-bound account issued OTP: %v", err)
	}
	if store.issued != 1 {
		t.Fatal("already-bound account caused OTP delivery")
	}
}
