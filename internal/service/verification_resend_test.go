package service

import (
	"context"
	"testing"

	"github.com/datvtph41107/bdspro/internal/config"
	db "github.com/datvtph41107/bdspro/internal/database"
	"github.com/jackc/pgx/v5"
)

type resendEmailStore struct {
	db.Store
	account db.Account
	calls   int
}

func (s *resendEmailStore) GetAccountByEmail(_ context.Context, email string) (db.GetAccountByEmailRow, error) {
	if email != s.account.Email {
		return db.GetAccountByEmailRow{}, pgx.ErrNoRows
	}
	return db.GetAccountByEmailRow{ID: s.account.ID, PersonID: s.account.PersonID, Email: s.account.Email, CreatedAt: s.account.CreatedAt, IsEmailVerified: s.account.IsEmailVerified}, nil
}
func (s *resendEmailStore) RecordEmailVerificationJob(_ context.Context, _ int64) error {
	s.calls++
	return nil
}

func TestResendEmailVerificationDoesNotEnumerateAccounts(t *testing.T) {
	ctx := context.Background()
	store := &resendEmailStore{account: db.Account{
		ID: 3, Email: "pending@example.test",
	}}
	svc := NewAuthService(store, nil, config.Config{})
	if err := svc.ResendVerificationEmail(ctx, "missing@example.test"); err != nil {
		t.Fatal(err)
	}
	if store.calls != 0 {
		t.Fatal("unknown email enqueued a job")
	}
	if err := svc.ResendVerificationEmail(ctx, "  PENDING@example.test  "); err != nil {
		t.Fatal(err)
	}
	if store.calls != 1 {
		t.Fatal("pending email did not enqueue verification")
	}
	store.account.IsEmailVerified = true
	if err := svc.ResendVerificationEmail(ctx, "pending@example.test"); err != nil {
		t.Fatal(err)
	}
	if store.calls != 1 {
		t.Fatal("already verified email enqueued a job")
	}
}
