package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/datvtph41107/bdspro/internal/config"
	db "github.com/datvtph41107/bdspro/internal/database"
	apperrors "github.com/datvtph41107/bdspro/internal/errors"
	"github.com/datvtph41107/bdspro/internal/token"
	"github.com/datvtph41107/bdspro/internal/util"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type sessionMemoryStore struct {
	db.Store
	account      db.Account
	passwordHash string
	session      db.Session
}

func (s *sessionMemoryStore) GetAccountByEmail(_ context.Context, email string) (db.GetAccountByEmailRow, error) {
	if !strings.EqualFold(email, s.account.Email) {
		return db.GetAccountByEmailRow{}, pgx.ErrNoRows
	}
	return db.GetAccountByEmailRow{ID: s.account.ID, PersonID: s.account.PersonID, Email: s.account.Email, CreatedAt: s.account.CreatedAt, IsEmailVerified: s.account.IsEmailVerified}, nil
}

func (s *sessionMemoryStore) GetPasswordHash(_ context.Context, id int64) (string, error) {
	if id != s.account.ID || s.passwordHash == "" {
		return "", pgx.ErrNoRows
	}
	return s.passwordHash, nil
}

func (s *sessionMemoryStore) CreateSession(_ context.Context, p db.CreateSessionParams) (db.Session, error) {
	s.session = db.Session{
		ID: p.ID, AccountID: p.AccountID, RefreshToken: p.RefreshToken,
		UserAgent: p.UserAgent, ClientIp: p.ClientIp, ExpiresAt: p.ExpiresAt,
	}
	return s.session, nil
}

func (s *sessionMemoryStore) GetSession(_ context.Context, id uuid.UUID) (db.Session, error) {
	if s.session.ID != id {
		return db.Session{}, pgx.ErrNoRows
	}
	return s.session, nil
}

func (s *sessionMemoryStore) BlockSession(_ context.Context, p db.BlockSessionParams) (db.Session, error) {
	if s.session.ID != p.ID || s.session.AccountID != p.AccountID ||
		s.session.RefreshToken != p.RefreshToken || s.session.IsBlocked {
		return db.Session{}, pgx.ErrNoRows
	}
	s.session.IsBlocked = true
	return s.session, nil
}

func TestAuthLoginRenewRevoke(t *testing.T) {
	hash, err := util.HashPassword("correct-password")
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{
		TokenSymmetricKey:    strings.Repeat("k", 32),
		AccessTokenDuration:  15 * time.Minute,
		RefreshTokenDuration: time.Hour,
	}
	maker, err := token.NewJWTMaker(cfg.TokenSymmetricKey)
	if err != nil {
		t.Fatal(err)
	}
	store := &sessionMemoryStore{account: db.Account{ID: 7, PersonID: 11, Email: "a@example.test"}, passwordHash: hash}
	svc := NewAuthService(store, maker, cfg)
	ctx := context.Background()

	if _, err := svc.Login(ctx, store.account.Email, "wrong", "", ""); !errors.Is(err, apperrors.InvalidCredentials) {
		t.Fatalf("invalid password: got %v", err)
	}

	result, err := svc.Login(ctx, store.account.Email, "correct-password", "test-client", "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := maker.VerifyToken(result.AccessToken, token.TokenTypeAccess); err != nil {
		t.Fatalf("access token invalid: %v", err)
	}
	if _, err := maker.VerifyToken(result.RefreshToken, token.TokenTypeAccess); err == nil {
		t.Fatal("refresh token must never authenticate as access token")
	}
	renewal, err := svc.Renew(ctx, result.RefreshToken)
	if err != nil || renewal.AccessToken == "" {
		t.Fatalf("renew failed: %v", err)
	}
	if err := svc.Revoke(ctx, result.RefreshToken); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Renew(ctx, result.RefreshToken); !errors.Is(err, apperrors.SessionUnavailable) {
		t.Fatalf("revoked session must not renew: %v", err)
	}
	// Existing access JWTs remain valid until expiry by current stateless policy.
	if _, err := maker.VerifyToken(result.AccessToken, token.TokenTypeAccess); err != nil {
		t.Fatalf("revocation must not silently change current JWT semantics: %v", err)
	}
}

type registrationStoreFake struct {
	db.Store
	calls int
}

func (s *registrationStoreFake) RegisterTx(_ context.Context, _ db.RegisterTxParams) (db.Account, error) {
	s.calls++
	return db.Account{ID: 7}, nil
}

func TestRegisterPasswordByteLimit(t *testing.T) {
	store := &registrationStoreFake{}
	svc := NewAuthService(store, nil, config.Config{})
	for _, password := range []string{
		"short",
		strings.Repeat("a", 73),
		strings.Repeat("😀", 19), // 19 runes, 76 bytes; Gin's max=72 alone would accept it.
	} {
		_, err := svc.Register(context.Background(), "test@example.test", password)
		if !errors.Is(err, apperrors.InvalidPassword) {
			t.Fatalf("password %q: expected invalid password, got %v", password, err)
		}
	}
	if store.calls != 0 {
		t.Fatalf("invalid passwords reached RegisterTx %d times", store.calls)
	}
	_, err := svc.Register(context.Background(), "test@example.test", strings.Repeat("a", 72))
	if err != nil || store.calls != 1 {
		t.Fatalf("72-byte password should be accepted: calls=%d err=%v", store.calls, err)
	}
}
