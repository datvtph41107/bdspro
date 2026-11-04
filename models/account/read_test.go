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
