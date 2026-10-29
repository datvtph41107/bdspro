package service

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/datvtph41107/bdspro/internal/config"
	db "github.com/datvtph41107/bdspro/internal/database"
	apperrors "github.com/datvtph41107/bdspro/internal/errors"
	"github.com/jackc/pgx/v5"
)

type linkPhoneProofStore struct {
	db.Store
	phone string
	err error
	callCount int
	gotAccount int64
	gotPhone string
	gotPurpose string
	gotMAC []byte
}

func (s *linkPhoneProofStore) GetPhoneForAccount(_ context.Context, accountID int64) (string, error) {
	s.gotAccount = accountID
	return s.phone, s.err
}
func (s *linkPhoneProofStore) ConsumeChallenge(_ context.Context, phone, purpose string, accountID int64, mac []byte) error {
	s.callCount++
	s.gotPhone = phone
	s.gotPurpose = purpose
	s.gotAccount = accountID
	s.gotMAC = append([]byte(nil), mac...)
	return s.err
}
func TestPhoneProofForGoogleLink(t *testing.T) {
	const phone = "+84901234567"
	const key = "01234567890123456789012345678901"
	store := &linkPhoneProofStore{phone: phone}
	s := PhoneOTP{Store: store, Config: config.Config{Environment: "development", TokenSymmetricKey: key}}
	if err := s.AuthorizeGoogleLink(context.Background(), 25, "123456"); err != nil {
		t.Fatal(err)
	}
	if store.callCount != 1 || store.gotPhone != phone || store.gotPurpose != "login" || store.gotAccount != 25 {
		t.Fatalf("wrong verification target: %+v", store)
	}
	if !bytes.Equal(store.gotMAC, s.mac(phone, "login", 25, "123456")) {
		t.Fatal("OTP verification did not bind account, phone and code")
	}
	store.callCount = 0
	if err := s.AuthorizeGoogleLink(context.Background(), 25, "incorrect"); !errors.Is(err, apperrors.OTPInvalid) {
		t.Fatalf("invalid phone code accepted: %v", err)
	}
	if store.callCount != 0 {
		t.Fatal("invalid code reached database")
	}
	store.err = pgx.ErrNoRows
	if err := s.AuthorizeGoogleLink(context.Background(), 25, "123456"); !errors.Is(err, apperrors.InvalidCredentials) {
		t.Fatalf("phone not bound to account: %v", err)
	}
}
