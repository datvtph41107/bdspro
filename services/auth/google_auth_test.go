package auth

import (
	"context"
	"errors"
	account_model "github.com/datvtph41107/bdspro/models/account"
	auth_model "github.com/datvtph41107/bdspro/models/auth"
	"strings"
	"testing"
	"time"

	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/modules/setting"
	"github.com/datvtph41107/bdspro/modules/token"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type googleStoreFake struct {
	db.Store
	linkErr    error
	lookupErr  error
	accountErr error
	createErr  error
	created    int
	sessions   int
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

func (s *googleStoreFake) GetAccountByID(_ context.Context, id int64) (db.Account, error) {
	if s.accountErr != nil {
		return db.Account{}, s.accountErr
	}
	return db.Account{ID: id, Email: "linked@example.test", IsEmailVerified: true}, nil
}

// This DBTX executes the generated SQLC calls in the model operation, not
// a second implementation of Google signup business rules.
type googleCreateDBTX struct{}

func (googleCreateDBTX) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.NewCommandTag("INSERT 0 1"), nil
}

func (googleCreateDBTX) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return nil, errors.New("unexpected SQLC query")
}

func (googleCreateDBTX) QueryRow(_ context.Context, statement string, args ...any) pgx.Row {
	return googleCreateRow{statement: statement, args: args}
}

type googleCreateRow struct {
	statement string
	args      []any
}

func (r googleCreateRow) Scan(dest ...any) error {
	switch {
	case strings.Contains(r.statement, "INSERT INTO parties DEFAULT VALUES"):
		*dest[0].(*int64) = 72
	case strings.Contains(r.statement, "INSERT INTO accounts"):
		*dest[0].(*int64) = 71
		*dest[1].(*int64) = 72
		*dest[2].(*string) = r.args[1].(string)
		*dest[3].(*time.Time) = time.Now()
		*dest[4].(*bool) = true
	case strings.Contains(r.statement, "INSERT INTO google_identities"):
		*dest[0].(*int64) = 71
	default:
		return errors.New("unexpected SQLC statement")
	}
	return nil
}

func (s *googleStoreFake) ExecTx(ctx context.Context, fn func(*db.Queries) error) error {
	s.created++
	if s.createErr != nil {
		return s.createErr
	}
	return fn(db.New(googleCreateDBTX{}))
}

func (s *googleStoreFake) CreateSession(_ context.Context, p db.CreateSessionParams) (db.Session, error) {
	s.sessions++
	return db.Session{
		ID: p.ID, AccountID: p.AccountID, RefreshToken: p.RefreshToken,
		UserAgent: p.UserAgent, ClientIp: p.ClientIp, ExpiresAt: p.ExpiresAt,
	}, nil
}

func googleTestService(store db.Store, t *testing.T) *GoogleAuthService {
	t.Helper()
	cfg := setting.Config{
		TokenSymmetricKey:    strings.Repeat("x", 32),
		AccessTokenDuration:  15 * time.Minute,
		RefreshTokenDuration: time.Hour,
	}
	maker, err := token.NewJWTMaker(cfg.TokenSymmetricKey)
	if err != nil {
		t.Fatal(err)
	}
	return NewGoogleAuthService(store, maker, cfg)
}

func TestGoogleLinkAndLoginErrorClassification(t *testing.T) {
	ctx := context.Background()
	store := &googleStoreFake{}
	auth := googleTestService(store, t)

	store.linkErr = pgx.ErrNoRows
	if err := auth.Link(ctx, 42, "google-sub"); !errors.Is(err, auth_model.ErrGoogleIdentityAlreadyLinked) {
		t.Fatalf("already-linked Google identity: %v", err)
	}
	store.linkErr = &pgconn.PgError{Code: "23505"}
	if err := auth.Link(ctx, 42, "google-sub"); !errors.Is(err, auth_model.ErrGoogleAccountAlreadyLinked) {
		t.Fatalf("duplicate linked account: %v", err)
	}
	store.linkErr = nil

	store.lookupErr = pgx.ErrNoRows
	unknown := VerifiedGoogleIdentity{Subject: "new-google-sub"}
	if _, err := auth.Login(ctx, unknown, "", ""); !errors.Is(err, auth_model.ErrGoogleEmailUnverified) {
		t.Fatalf("unverified email must not create account: %v", err)
	}
	if store.created != 0 {
		t.Fatal("unverified email caused account creation")
	}

	verified := VerifiedGoogleIdentity{Subject: uuid.NewString(), Email: "New@Example.Test", EmailVerified: true}
	result, err := auth.Login(ctx, verified, "agent", "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	if store.created != 1 || store.sessions != 1 || result.Account.ID != 71 ||
		result.Account.Email != "new@example.test" || !result.Account.IsEmailVerified {
		t.Fatalf("incorrect Google account/session: %+v, created=%d sessions=%d", result.Account, store.created, store.sessions)
	}
	store.createErr = &pgconn.PgError{Code: "23505", ConstraintName: "accounts_email_login_uq"}
	if _, err := auth.Login(ctx, verified, "", ""); !errors.Is(err, auth_model.ErrGoogleEmailAlreadyRegistered) {
		t.Fatalf("existing email must require explicit linking: %v", err)
	}
	store.createErr = nil

	store.lookupErr = nil
	store.accountErr = pgx.ErrNoRows
	if _, err := auth.Login(ctx, verified, "", ""); !errors.Is(err, account_model.ErrAccountUnavailable) {
		t.Fatalf("Google subject linked to deleted account: %v", err)
	}
	if err := auth.CheckPassword(ctx, 42, "password"); !errors.Is(err, auth_model.ErrInvalidCredentials) {
		t.Fatalf("removed account cannot link Google: %v", err)
	}
}
