package service

import (
	"context"
	"errors"
	"testing"

	"github.com/datvtph41107/bdspro/internal/config"
	apperrors "github.com/datvtph41107/bdspro/internal/core/errors"
	db "github.com/datvtph41107/bdspro/internal/db/sqlc"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type googleStoreFake struct {
	db.Store
	linkErr error
	lookupErr error
	accountErr error
}

func (s *googleStoreFake) LinkGoogleIdentity(_ context.Context, _ db.LinkGoogleIdentityParams) error {
	return s.linkErr
}

func (s *googleStoreFake) GetGoogleAccountID(_ context.Context, _ string) (int64, error) {
	if s.lookupErr != nil {
		return 0, s.lookupErr
	}
	return 42, nil
}

func (s *googleStoreFake) GetAccountByID(_ context.Context, _ int64) (db.Account, error) {
	if s.accountErr != nil {
		return db.Account{}, s.accountErr
	}
	return db.Account{ID: 42}, nil
}

func TestGoogleLinkAndLoginErrorClassification(t *testing.T) {
	ctx := context.Background()
	store := &googleStoreFake{}
	auth := NewGoogleAuthService(store, nil, config.Config{})

	store.linkErr = pgx.ErrNoRows
	if err := auth.Link(ctx, 42, "google-sub"); !errors.Is(err, apperrors.GoogleIdentityAlreadyLinked) {
		t.Fatalf("already-linked Google identity: %v", err)
	}
	store.linkErr = &pgconn.PgError{Code: "23505"}
	if err := auth.Link(ctx, 42, "google-sub"); !errors.Is(err, apperrors.GoogleAccountAlreadyLinked) {
		t.Fatalf("duplicate linked account: %v", err)
	}
	store.linkErr = nil

	store.lookupErr = pgx.ErrNoRows
	if _, err := auth.Login(ctx, "not-linked", "", ""); !errors.Is(err, apperrors.GoogleIdentityUnlinked) {
		t.Fatalf("unknown Google subject: %v", err)
	}
	store.lookupErr = nil

	store.accountErr = pgx.ErrNoRows
	if _, err := auth.Login(ctx, "google-sub", "", ""); !errors.Is(err, apperrors.AccountUnavailable) {
		t.Fatalf("Google subject with removed account: %v", err)
	}
	if err := auth.CheckPassword(ctx, 42, "password"); !errors.Is(err, apperrors.InvalidCredentials) {
		t.Fatalf("cannot link using removed account: %v", err)
	}
}
