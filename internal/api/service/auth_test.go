package service

import (
    "context"
    "errors"
    "strings"
    "testing"
    "time"

    "github.com/datvtph41107/bdspro/internal/config"
    apperrors "github.com/datvtph41107/bdspro/internal/core/errors"
    db "github.com/datvtph41107/bdspro/internal/db/sqlc"
    "github.com/datvtph41107/bdspro/internal/token"
    "github.com/datvtph41107/bdspro/internal/util"
    "github.com/google/uuid"
    "github.com/jackc/pgx/v5"
)

type sessionMemoryStore struct {
    db.Store
    account db.Account
    session db.Session
}

func (s *sessionMemoryStore) GetAccountByEmail(_ context.Context, email string) (db.Account, error) {
    if !strings.EqualFold(email, s.account.Email) {
        return db.Account{}, pgx.ErrNoRows
    }
    return s.account, nil
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
        TokenSymmetricKey: strings.Repeat("k", 32),
        AccessTokenDuration: 15 * time.Minute,
        RefreshTokenDuration: time.Hour,
    }
    maker, err := token.NewJWTMaker(cfg.TokenSymmetricKey)
    if err != nil {
        t.Fatal(err)
    }
    store := &sessionMemoryStore{account: db.Account{ID: 7, PersonID: 11, Email: "a@example.test", PasswordHash: hash}}
    svc := NewAuthService(store, nil, maker, cfg)
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
