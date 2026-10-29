package jobs

import (
	"context"
	"errors"
	"strings"
	"testing"

	db "github.com/datvtph41107/bdspro/internal/database"
	"github.com/jackc/pgx/v5"
)

type verificationStoreFake struct {
	account    db.Account
	record     db.VerifyEmail
	accountErr error
	tokenErr   error
	tokens     int
}

func (s *verificationStoreFake) GetAccountByID(_ context.Context, _ int64) (db.GetAccountByIDRow, error) {
	if s.accountErr != nil {
		return db.GetAccountByIDRow{}, s.accountErr
	}
	return db.GetAccountByIDRow{ID: s.account.ID, PersonID: s.account.PersonID, Email: s.account.Email, CreatedAt: s.account.CreatedAt, IsEmailVerified: s.account.IsEmailVerified}, nil
}

func (s *verificationStoreFake) GetOrCreateVerifyEmailTx(_ context.Context, _ db.CreateVerifyEmailParams) (db.VerifyEmail, error) {
	s.tokens++
	if s.tokenErr != nil {
		return db.VerifyEmail{}, s.tokenErr
	}
	return s.record, nil
}

type fakeMailSender struct {
	fail    error
	content []string
}

func (m *fakeMailSender) SendEmail(_, content string, _, _, _, _ []string) error {
	if m.fail != nil {
		return m.fail
	}
	m.content = append(m.content, content)
	return nil
}

func TestVerificationJobSkipsAlreadyVerifiedAccount(t *testing.T) {
	store := &verificationStoreFake{account: db.Account{ID: 8, IsEmailVerified: true}}
	sender := &fakeMailSender{}
	if err := SendVerificationEmail(context.Background(), store, sender, "https://example.test/verify", 8); err != nil {
		t.Fatal(err)
	}
	if store.tokens != 0 || len(sender.content) != 0 {
		t.Fatal("verified account must not request a token or email")
	}
}

func TestVerificationJobRetriesWithReusableCode(t *testing.T) {
	store := &verificationStoreFake{
		account: db.Account{ID: 8, Email: "example@example.test"},
		record:  db.VerifyEmail{ID: 20, AccountID: 8, SecretCode: "reused-secret"},
	}
	sender := &fakeMailSender{fail: errors.New("SMTP unavailable")}
	ctx := context.Background()
	url := "https://example.test/users/verify_email"
	if err := SendVerificationEmail(ctx, store, sender, url, 8); err == nil {
		t.Fatal("SMTP failure must remain retryable")
	}
	sender.fail = nil
	if err := SendVerificationEmail(ctx, store, sender, url, 8); err != nil {
		t.Fatal(err)
	}
	if store.tokens != 2 || len(sender.content) != 1 ||
		!strings.Contains(sender.content[0], "reused-secret") {
		t.Fatal("retry did not reuse persisted verification token")
	}
	store.accountErr = pgx.ErrNoRows
	if err := SendVerificationEmail(ctx, store, sender, url, 8); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("missing account must remain retryable: %v", err)
	}
}
