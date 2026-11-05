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

type linkPhoneProofStore struct {
	db.Store
	phone      string
	err        error
	callCount  int
	gotAccount int64
	gotPhone   string
	gotPurpose string
	gotMAC     []byte
}

func (s *linkPhoneProofStore) GetPhoneForAccount(_ context.Context, accountID int64) (string, error) {
	s.gotAccount = accountID
	return s.phone, s.err
}
func (s *linkPhoneProofStore) ExecTx(_ context.Context, _ func(*db.Queries) error) error {
	s.callCount++
	return s.err
}
func TestPhoneProofForOAuth2CredentialLink(t *testing.T) {
	const phone = "+84901234567"
	const key = "01234567890123456789012345678901"
	store := &linkPhoneProofStore{phone: phone}
	s := NewAuthService(store, nil, setting.Config{Environment: "development", TokenSymmetricKey: key})
	if err := s.AuthorizePhoneProof(context.Background(), 25, "123456"); err != nil {
		t.Fatal(err)
	}
	if store.callCount != 1 || store.gotAccount != 25 {
		t.Fatalf("phone proof did not reach transaction boundary: %+v", store)
	}
	store.callCount = 0
	if err := s.AuthorizePhoneProof(context.Background(), 25, "incorrect"); !errors.Is(err, auth_model.ErrOTPInvalid) {
		t.Fatalf("invalid phone code accepted: %v", err)
	}
	if store.callCount != 0 {
		t.Fatal("invalid code reached database")
	}
	store.err = pgx.ErrNoRows
	if err := s.AuthorizePhoneProof(context.Background(), 25, "123456"); !errors.Is(err, auth_model.ErrInvalidCredentials) {
		t.Fatalf("phone not bound to account: %v", err)
	}
}
