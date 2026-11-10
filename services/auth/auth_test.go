package auth

import (
	"context"
	"errors"
	"fmt"
	account_model "github.com/datvtph41107/bdspro/models/account"
	audit_model "github.com/datvtph41107/bdspro/models/audit"
	auth_model "github.com/datvtph41107/bdspro/models/auth"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	db "github.com/datvtph41107/bdspro/models/db"
	authcrypto "github.com/datvtph41107/bdspro/modules/auth"
	"github.com/datvtph41107/bdspro/modules/setting"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type sessionMemoryStore struct {
	db.Store
	account             db.Account
	passwordHash        string
	session             db.Session
	historyCount        int
	historyErr          error
	beforeSessionInsert func(*sessionMemoryStore)
	auditErr            error
	auditEvents         []db.InsertAuditEventParams
}

func (s *sessionMemoryStore) InsertAuditEvent(_ context.Context, arg db.InsertAuditEventParams) error {
	if s.auditErr != nil {
		return s.auditErr
	}
	s.auditEvents = append(s.auditEvents, arg)
	return nil
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

// sessionMemoryDBTX provides only the SQLC operations exercised by NewSession.
// It is intentionally transaction-scoped; no production test bypass is installed.
type sessionMemoryDBTX struct {
	store *sessionMemoryStore
}

type sessionMemoryRow struct {
	session db.Session
	err     error
}

func (r sessionMemoryRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	if len(dest) != 8 {
		return fmt.Errorf("unexpected session column count: %d", len(dest))
	}
	*dest[0].(*uuid.UUID) = r.session.ID
	*dest[1].(*int64) = r.session.AccountID
	*dest[2].(*string) = r.session.RefreshToken
	*dest[3].(*string) = r.session.UserAgent
	*dest[4].(*string) = r.session.ClientIp
	*dest[5].(*bool) = r.session.IsBlocked
	*dest[6].(*time.Time) = r.session.CreatedAt
	*dest[7].(*time.Time) = r.session.ExpiresAt
	return nil
}

type sessionMemoryPasswordRow struct {
	hash string
}

func (r sessionMemoryPasswordRow) Scan(dest ...any) error {
	if len(dest) != 1 {
		return fmt.Errorf("unexpected credential query shape")
	}
	*dest[0].(*string) = r.hash
	return nil
}

func (tx sessionMemoryDBTX) QueryRow(_ context.Context, query string, args ...any) pgx.Row {
	if strings.Contains(query, "SELECT password_hash") && len(args) == 1 {
		if args[0].(int64) != tx.store.account.ID || tx.store.passwordHash == "" {
			return sessionMemoryRow{err: pgx.ErrNoRows}
		}
		return sessionMemoryPasswordRow{hash: tx.store.passwordHash}
	}
	if !strings.Contains(query, "INSERT INTO sessions") || len(args) != 7 {
		return sessionMemoryRow{err: fmt.Errorf("unexpected session query")}
	}
	if tx.store.beforeSessionInsert != nil {
		tx.store.beforeSessionInsert(tx.store)
	}
	s := db.Session{
		ID: args[0].(uuid.UUID), AccountID: args[1].(int64),
		RefreshToken: args[2].(string), UserAgent: args[3].(string),
		ClientIp: args[4].(string), IsBlocked: args[5].(bool),
		ExpiresAt: args[6].(time.Time), CreatedAt: time.Now(),
	}
	tx.store.session = s
	return sessionMemoryRow{session: s}
}

func (tx sessionMemoryDBTX) Exec(_ context.Context, query string, _ ...any) (pgconn.CommandTag, error) {
	if !strings.Contains(query, "INSERT INTO account_connection_histories") {
		return pgconn.CommandTag{}, fmt.Errorf("unexpected transaction write")
	}
	if tx.store.historyErr != nil {
		return pgconn.CommandTag{}, tx.store.historyErr
	}
	tx.store.historyCount++
	return pgconn.NewCommandTag("INSERT 0 1"), nil
}

func (tx sessionMemoryDBTX) Query(_ context.Context, _ string, _ ...any) (pgx.Rows, error) {
	return nil, fmt.Errorf("unexpected transaction read")
}

func (s *sessionMemoryStore) ExecTx(ctx context.Context, fn func(*db.Queries) error) error {
	before, count := s.session, s.historyCount
	if err := fn(db.New(sessionMemoryDBTX{store: s})); err != nil {
		s.session, s.historyCount = before, count
		return err
	}
	return nil
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
	hash, err := authcrypto.HashPassword("correct-password")
	if err != nil {
		t.Fatal(err)
	}
	cfg := setting.Config{
		TokenSymmetricKey:    strings.Repeat("k", 32),
		AuditKey:             strings.Repeat("a", 32),
		AccessTokenDuration:  15 * time.Minute,
		RefreshTokenDuration: time.Hour,
	}
	maker, err := authcrypto.NewJWTMaker(cfg.TokenSymmetricKey)
	if err != nil {
		t.Fatal(err)
	}
	store := &sessionMemoryStore{account: db.Account{ID: 7, PersonID: 11, Email: "a@example.test"}, passwordHash: hash}
	svc := NewAuthService(store, maker, cfg, nil)
	ctx := context.Background()

	if _, err := svc.Login(ctx, store.account.Email, "wrong", "", "203.0.113.9"); !errors.Is(err, auth_model.ErrInvalidCredentials) {
		t.Fatalf("invalid password: got %v", err)
	}
	if len(store.auditEvents) != 1 || store.auditEvents[0].Reason.String != audit_model.ReasonInvalidPassword ||
		!store.auditEvents[0].AccountID.Valid || store.auditEvents[0].AccountID.Int64 != store.account.ID {
		t.Fatalf("rejected known credential not persisted with account scope: %+v", store.auditEvents)
	}

	result, err := svc.Login(ctx, store.account.Email, "correct-password", "test-client", "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	accessPayload, err := maker.VerifyToken(result.AccessToken, authcrypto.TokenTypeAccess)
	if err != nil || accessPayload.SessionID != result.Session.ID {
		t.Fatalf("access token not bound to its persisted session: %+v %v", accessPayload, err)
	}
	if _, err := maker.VerifyToken(result.RefreshToken, authcrypto.TokenTypeAccess); err == nil {
		t.Fatal("refresh token must never authenticate as access token")
	}
	if store.historyCount != 1 {
		t.Fatalf("session issuance must produce one login history, got %d", store.historyCount)
	}
	if len(store.auditEvents) != 1 {
		t.Fatalf("successful session issuance must not create a rejected-proof event: %+v", store.auditEvents)
	}
	if store.session.RefreshToken != auth_model.RefreshTokenDigest(result.RefreshToken) || store.session.RefreshToken == result.RefreshToken {
		t.Fatal("raw refresh JWT persisted in session storage")
	}
	renewal, err := svc.Renew(ctx, result.RefreshToken)
	if err != nil || renewal.AccessToken == "" {
		t.Fatalf("renew failed: %v", err)
	}
	renewedPayload, err := maker.VerifyToken(renewal.AccessToken, authcrypto.TokenTypeAccess)
	if err != nil || renewedPayload.SessionID != result.Session.ID {
		t.Fatalf("renewal lost original session authority: %+v %v", renewedPayload, err)
	}
	if err := svc.Revoke(ctx, result.RefreshToken); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Renew(ctx, result.RefreshToken); !errors.Is(err, auth_model.ErrSessionUnavailable) {
		t.Fatalf("revoked session must not renew: %v", err)
	}
	// Existing access JWTs remain valid until expiry by current stateless policy.
	if _, err := maker.VerifyToken(result.AccessToken, authcrypto.TokenTypeAccess); err != nil {
		t.Fatalf("revocation must not silently change current JWT semantics: %v", err)
	}
}

type registrationStoreFake struct {
	db.Store
	calls int
}

func (s *registrationStoreFake) ExecTx(_ context.Context, _ func(*db.Queries) error) error {
	s.calls++
	return nil
}

func TestRegisterPasswordByteLimit(t *testing.T) {
	store := &registrationStoreFake{}
	svc := NewAuthService(store, nil, setting.Config{}, nil)
	for _, password := range []string{
		"short",
		strings.Repeat("a", 73),
		strings.Repeat("😀", 19), // 19 runes, 76 bytes; Gin's max=72 alone would accept it.
	} {
		_, err := svc.Register(context.Background(), "test@example.test", password)
		if !errors.Is(err, auth_model.ErrInvalidPassword) {
			t.Fatalf("password %q: expected invalid password, got %v", password, err)
		}
	}
	if store.calls != 0 {
		t.Fatalf("invalid passwords reached account transaction %d times", store.calls)
	}
	_, err := svc.Register(context.Background(), "test@example.test", strings.Repeat("a", 72))
	if err != nil || store.calls != 1 {
		t.Fatalf("72-byte password should be accepted: calls=%d err=%v", store.calls, err)
	}
}

// Legacy refresh sessions remain valid only for the configured expiry window.
// This test exercises the same renewal and revocation APIs across the rollout.
func TestAuthLegacyRefreshSessionTransition(t *testing.T) {
	cfg := setting.Config{
		TokenSymmetricKey:    strings.Repeat("k", 32),
		AuditKey:             strings.Repeat("a", 32),
		AccessTokenDuration:  15 * time.Minute,
		RefreshTokenDuration: time.Hour,
	}
	maker, err := authcrypto.NewJWTMaker(cfg.TokenSymmetricKey)
	if err != nil {
		t.Fatal(err)
	}
	store := &sessionMemoryStore{account: db.Account{ID: 17}}
	svc := NewAuthService(store, maker, cfg, nil)
	issued, err := svc.issueSession(context.Background(), account_model.AccountFromDB(store.account), "", "")
	if err != nil {
		t.Fatal(err)
	}
	store.session.RefreshToken = issued.RefreshToken // simulates legacy DB row
	renewal, err := svc.Renew(context.Background(), issued.RefreshToken)
	if err != nil || renewal.AccessToken == "" {
		t.Fatalf("legacy token could not renew: %v", err)
	}
	if err := svc.Revoke(context.Background(), issued.RefreshToken); err != nil {
		t.Fatalf("legacy token could not revoke: %v", err)
	}
	if _, err := svc.Renew(context.Background(), issued.RefreshToken); !errors.Is(err, auth_model.ErrSessionUnavailable) {
		t.Fatalf("revoked legacy refresh proof renewed: %v", err)
	}
}

func TestRejectedProofAuditFailureKeepsInvalidCredentialResponse(t *testing.T) {
	hash, err := authcrypto.HashPassword("correct-password")
	if err != nil {
		t.Fatal(err)
	}
	store := &sessionMemoryStore{
		account:      db.Account{ID: 44, Email: "known@example.test"},
		passwordHash: hash,
		auditErr:     errors.New("authentication event database unavailable"),
	}
	svc := NewAuthService(store, nil, setting.Config{
		TokenSymmetricKey: strings.Repeat("k", 32),
		AuditKey:          strings.Repeat("a", 32),
	}, nil)
	_, err = svc.Login(context.Background(), "known@example.test", "wrong", "", "203.0.113.9")
	if !errors.Is(err, auth_model.ErrInvalidCredentials) {
		t.Fatalf("best-effort rejection audit must not change public error: %v", err)
	}
	if store.session.ID != uuid.Nil || len(store.auditEvents) != 0 {
		t.Fatalf("failed rejection must never issue a session: %+v", store.session)
	}
}

func TestRejectedOIDCIdentityAuditDoesNotGrantSession(t *testing.T) {
	store := &sessionMemoryStore{}
	svc := NewAuthService(store, nil, setting.Config{
		TokenSymmetricKey: strings.Repeat("k", 32),
		AuditKey:          strings.Repeat("a", 32),
	}, nil)
	_, err := svc.LoginOIDC(context.Background(), auth_model.OIDCClaims{
		Issuer:  "http://untrusted-issuer.example.test",
		Subject: "subject-a",
	}, "", "203.0.113.8")
	if !errors.Is(err, auth_model.ErrOIDCIssuerInvalid) {
		t.Fatalf("invalid issuer must remain rejected: %v", err)
	}
	if len(store.auditEvents) != 1 || store.auditEvents[0].Method.String != audit_model.MethodOIDC ||
		store.auditEvents[0].Reason.String != audit_model.ReasonInvalidIdentity || store.auditEvents[0].AccountID.Valid {
		t.Fatalf("invalid OIDC decision missing or misattributed: %+v", store.auditEvents)
	}
	if store.session.ID != uuid.Nil {
		t.Fatal("invalid identity issued a session")
	}
}

func TestPhoneLoginInvalidProofWritesNoSession(t *testing.T) {
	store := &sessionMemoryStore{}
	svc := NewAuthService(store, nil, setting.Config{
		TokenSymmetricKey: strings.Repeat("k", 32),
		AuditKey:          strings.Repeat("a", 32),
		Environment:       "development",
	}, nil)
	_, err := svc.VerifyPhoneLogin(context.Background(), "+84901001001", "wrong-code", "", "203.0.113.10")
	if !errors.Is(err, auth_model.ErrOTPInvalid) {
		t.Fatalf("invalid OTP input must not authenticate: %v", err)
	}
	if len(store.auditEvents) != 1 || store.auditEvents[0].Method.String != audit_model.MethodPhoneOTP ||
		store.auditEvents[0].Reason.String != audit_model.ReasonInvalidOTP {
		t.Fatalf("bad OTP was not recorded: %+v", store.auditEvents)
	}
	if store.session.ID != uuid.Nil {
		t.Fatal("bad phone OTP issued a session")
	}
}

func TestSessionHistoryFailureDoesNotReleaseCredentials(t *testing.T) {
	maker, err := authcrypto.NewJWTMaker(strings.Repeat("k", 32))
	if err != nil {
		t.Fatal(err)
	}
	store := &sessionMemoryStore{
		account:    db.Account{ID: 31},
		historyErr: errors.New("audit insertion unavailable"),
	}
	svc := NewAuthService(store, maker, setting.Config{
		TokenSymmetricKey:    strings.Repeat("k", 32),
		AuditKey:             strings.Repeat("a", 32),
		AccessTokenDuration:  time.Minute,
		RefreshTokenDuration: time.Hour,
	}, nil)
	result, err := svc.issueSession(context.Background(), account_model.AccountFromDB(store.account), "test agent", "203.0.113.1")
	if err == nil || result.RefreshToken != "" || result.AccessToken != "" {
		t.Fatalf("history write failure released a token: %+v err=%v", result, err)
	}
	if store.session.ID != uuid.Nil || store.historyCount != 0 {
		t.Fatalf("failed transaction persisted session=%+v histories=%d", store.session, store.historyCount)
	}
}

func TestSessionUserAgentNormalizedBeforePersistence(t *testing.T) {
	maker, err := authcrypto.NewJWTMaker(strings.Repeat("k", 32))
	if err != nil {
		t.Fatal(err)
	}
	store := &sessionMemoryStore{account: db.Account{ID: 32}}
	svc := NewAuthService(store, maker, setting.Config{
		TokenSymmetricKey:    strings.Repeat("k", 32),
		AuditKey:             strings.Repeat("a", 32),
		AccessTokenDuration:  time.Minute,
		RefreshTokenDuration: time.Hour,
	}, nil)
	userAgent := strings.Repeat("✓", 2050) + "\x00" + string([]byte{0xff})
	if _, err := svc.issueSession(context.Background(), account_model.AccountFromDB(store.account), userAgent, "203.0.113.2"); err != nil {
		t.Fatal(err)
	}
	if n := utf8.RuneCountInString(store.session.UserAgent); n != 2048 {
		t.Fatalf("user agent length = %d, want 2048", n)
	}
	if !utf8.ValidString(store.session.UserAgent) || strings.ContainsRune(store.session.UserAgent, '\x00') {
		t.Fatal("invalid or database-disallowed user-agent persisted")
	}
	if store.historyCount != 1 {
		t.Fatalf("successful session should have one history, got %d", store.historyCount)
	}
}

func TestLoginRejectsStaleVerifiedPasswordHashInsideSessionTransaction(t *testing.T) {
	old, err := authcrypto.HashPassword("old-password")
	if err != nil {
		t.Fatal(err)
	}
	newHash, err := authcrypto.HashPassword("new-password")
	if err != nil {
		t.Fatal(err)
	}
	store := &sessionMemoryStore{account: db.Account{ID: 77, PersonID: 88, Email: "stale@example.test"}, passwordHash: old}
	cfg := setting.Config{TokenSymmetricKey: strings.Repeat("k", 32), AuditKey: strings.Repeat("a", 32),
		AccessTokenDuration: 15 * time.Minute, RefreshTokenDuration: time.Hour}
	maker, err := authcrypto.NewJWTMaker(cfg.TokenSymmetricKey)
	if err != nil {
		t.Fatal(err)
	}
	// Password was verified against 'old'; recovery commits a new password
	// before session issuance reaches the transaction.
	store.passwordHash = newHash
	_, err = auth_model.NewSession(context.Background(), store, db.CreateSessionParams{
		ID: uuid.New(), AccountID: 77, RefreshToken: "stale-test",
		UserAgent: "stale-test", ClientIp: "127.0.0.1", ExpiresAt: time.Now().Add(time.Hour),
	}, old)
	if !errors.Is(err, auth_model.ErrInvalidCredentials) {
		t.Fatalf("stale password issued a session: %v", err)
	}
	if store.session.ID != uuid.Nil || store.historyCount != 0 {
		t.Fatal("stale credential persisted session/history")
	}
	// End-to-end service call: after bcrypt succeeds but before the session
	// is inserted, a recovery transitions the credential to a new hash.
	store.passwordHash = old
	store.beforeSessionInsert = func(m *sessionMemoryStore) {
		m.passwordHash = newHash
	}
	service := NewAuthService(store, maker, cfg, nil)
	if _, err := service.Login(context.Background(), store.account.Email, "old-password", "agent", "127.0.0.1"); !errors.Is(err, auth_model.ErrInvalidCredentials) {
		t.Fatalf("late password reset did not abort login: %v", err)
	}
	if store.session.ID != uuid.Nil || store.historyCount != 0 {
		t.Fatal("late password reset left session/history writes")
	}
}
