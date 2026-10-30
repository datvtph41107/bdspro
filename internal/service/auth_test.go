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
	loginEvents  []db.RecordPasswordLoginEventParams
	auditErr     error
}

func (s *sessionMemoryStore) GetAccountByEmail(_ context.Context, email string) (db.Account, error) {
	if !strings.EqualFold(email, s.account.Email) {
		return db.Account{}, pgx.ErrNoRows
	}
	return s.account, nil
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

func (s *sessionMemoryStore) RecordPasswordLoginEvent(_ context.Context, arg db.RecordPasswordLoginEventParams) error {
	if s.auditErr != nil {
		return s.auditErr
	}
	s.loginEvents = append(s.loginEvents, arg)
	return nil
}

func (s *sessionMemoryStore) ListOwnLoginEvents(_ context.Context, accountID int64) ([]db.ListOwnLoginEventsRow, error) {
	var events []db.ListOwnLoginEventsRow
	for i, event := range s.loginEvents {
		if event.AccountID.Valid && event.AccountID.Int64 == accountID {
			events = append(events, db.ListOwnLoginEventsRow{ID: int64(i + 1), Outcome: event.Outcome})
		}
	}
	return events, nil
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

func TestPasswordLoginAuditBoundaries(t *testing.T) {
	hash, err := util.HashPassword("correct-password")
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{
		TokenSymmetricKey: strings.Repeat("k", 32),
		AccessTokenDuration: 15 * time.Minute,
		RefreshTokenDuration: time.Hour,
	}
	maker, err := token.NewJWTMaker(cfg.TokenSymmetricKey)
	if err != nil {
		t.Fatal(err)
	}
	store := &sessionMemoryStore{
		account: db.Account{ID: 71, Email: "known@example.test"},
		passwordHash: hash,
	}
	service := NewAuthService(store, maker, cfg)
	ctx := context.Background()
	for _, email := range []string{"other@example.test", "known@example.test"} {
		_, err := service.Login(ctx, email, "bad-password", strings.Repeat("u", 600), "192.0.2.10")
		if !errors.Is(err, apperrors.InvalidCredentials) {
			t.Fatalf("login error for %s = %v", email, err)
		}
	}
	if len(store.loginEvents) != 2 || store.loginEvents[0].AccountID.Valid ||
		!store.loginEvents[1].AccountID.Valid || store.loginEvents[1].AccountID.Int64 != 71 ||
		len([]rune(store.loginEvents[1].UserAgent)) != 512 {
		t.Fatalf("unexpected audit events: %+v", store.loginEvents)
	}
	if _, err := service.Login(ctx, "known@example.test", "correct-password", "test", "192.0.2.11"); err != nil {
		t.Fatal(err)
	}
	if len(store.loginEvents) != 3 || store.loginEvents[2].Outcome != "succeeded" {
		t.Fatalf("success not audited: %+v", store.loginEvents)
	}
	own, err := service.LoginHistory(ctx, 71)
	if err != nil || len(own) != 2 {
		t.Fatalf("own event history = %+v, %v", own, err)
	}
	other, err := service.LoginHistory(ctx, 99)
	if err != nil || len(other) != 0 {
		t.Fatalf("cross-account audit history leaked: %+v, %v", other, err)
	}
}

func TestPasswordLoginFailsClosedWhenAuditUnavailable(t *testing.T) {
	hash, err := util.HashPassword("correct-password")
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{TokenSymmetricKey: strings.Repeat("k", 32), AccessTokenDuration: time.Minute, RefreshTokenDuration: time.Hour}
	maker, err := token.NewJWTMaker(cfg.TokenSymmetricKey)
	if err != nil {
		t.Fatal(err)
	}
	store := &sessionMemoryStore{
		account: db.Account{ID: 15, Email: "known@example.test"},
		passwordHash: hash,
		auditErr: errors.New("audit database offline"),
	}
	service := NewAuthService(store, maker, cfg)
	_, err = service.Login(context.Background(), "known@example.test", "correct-password", "", "")
	if err == nil || !store.session.IsBlocked {
		t.Fatalf("login returned credentials or left refresh session active: %v", err)
	}
	_, err = service.Login(context.Background(), "other@example.test", "x", "", "")
	if err == nil || errors.Is(err, apperrors.InvalidCredentials) {
		t.Fatalf("audit outage returned an ordinary 401: %v", err)
	}
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
