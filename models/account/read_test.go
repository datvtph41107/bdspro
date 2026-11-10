package account

import (
	"context"
	"errors"
	"testing"

	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/jackc/pgx/v5"
)

type accountReaderFake struct {
	calls int
	value db.Account
	err   error
}

func (s *accountReaderFake) GetAccountByID(_ context.Context, _ int64) (db.Account, error) {
	s.calls++
	return s.value, s.err
}

func (s *accountReaderFake) GetAccountByEmail(_ context.Context, _ string) (db.Account, error) {
	s.calls++
	return s.value, s.err
}

func TestGetByID(t *testing.T) {
	ctx := context.Background()
	r := &accountReaderFake{value: db.Account{ID: 23, Email: "person@example.test"}}
	if _, err := GetByID(ctx, r, 0); !errors.Is(err, ErrAccountUnavailable) || r.calls != 0 {
		t.Fatalf("invalid identity must not query database: calls=%d err=%v", r.calls, err)
	}
	got, err := GetByID(ctx, r, 23)
	if err != nil || got.ID != 23 || got.Email != "person@example.test" {
		t.Fatalf("live account read: %+v / %v", got, err)
	}
	r.err = pgx.ErrNoRows
	if _, err := GetByID(ctx, r, 23); !errors.Is(err, ErrAccountUnavailable) {
		t.Fatalf("missing account must become canonical identity error: %v", err)
	}
	dbFailure := errors.New("private database connection diagnostics")
	r.err = dbFailure
	if _, err := GetByID(ctx, r, 23); !errors.Is(err, dbFailure) {
		t.Fatalf("unknown driver error must remain private and unclassified: %v", err)
	}
}

func TestGetByEmailUsesCanonicalAccountProjection(t *testing.T) {
	ctx := context.Background()
	r := &accountReaderFake{value: db.Account{
		ID: 23, PersonID: 24, Email: "person@example.test",
		PasswordResetVersion: 49,
	}}
	got, err := GetByEmail(ctx, r, "person@example.test")
	if err != nil || got == nil || got.ID != r.value.ID ||
		got.PersonID != r.value.PersonID || got.Email != r.value.Email {
		t.Fatalf("canonical email account read failed: %+v / %v", got, err)
	}
	if r.calls != 1 {
		t.Fatalf("lookup executed %d times", r.calls)
	}
	r.err = pgx.ErrNoRows
	if missing, err := GetByEmail(ctx, r, "missing@example.test"); missing != nil ||
		!errors.Is(err, ErrAccountUnavailable) {
		t.Fatalf("missing email should be a domain error, got %+v / %v", missing, err)
	}
	privateFailure := errors.New("private DB diagnostics")
	r.err = privateFailure
	if missing, err := GetByEmail(ctx, r, "person@example.test"); missing != nil ||
		!errors.Is(err, privateFailure) {
		t.Fatalf("DB failure was disguised, got %+v / %v", missing, err)
	}
}
