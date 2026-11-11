package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"html"
	"math/big"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	account_model "github.com/datvtph41107/bdspro/models/account"
	audit_model "github.com/datvtph41107/bdspro/models/audit"
	auth_model "github.com/datvtph41107/bdspro/models/auth"
	db "github.com/datvtph41107/bdspro/models/db"
	authcrypto "github.com/datvtph41107/bdspro/modules/auth"
	"github.com/datvtph41107/bdspro/modules/log"
	"github.com/datvtph41107/bdspro/modules/setting"
	"github.com/datvtph41107/bdspro/modules/validation"
	"github.com/datvtph41107/bdspro/services/mail"
	"github.com/datvtph41107/bdspro/services/notification"
	"github.com/jackc/pgx/v5"
)

// AuthService owns registration, credential authentication, email verification,
// access renewal and session revocation. No Gin or HTTP response types enter
// these operations.
type AuthService struct {
	maker       authcrypto.Maker
	config      setting.Config
	sender      notification.PhoneCodeSender
	emailSender mail.EmailSender
}

func NewAuthService(maker authcrypto.Maker, cfg setting.Config, sender notification.PhoneCodeSender, emailSender ...mail.EmailSender) *AuthService {
	s := &AuthService{maker: maker, config: cfg, sender: sender}
	if len(emailSender) > 0 {
		s.emailSender = emailSender[0]
	}
	return s
}

// HasPhone indicates that the HTTP phone channel is configured.
func (s *AuthService) HasPhone() bool { return s != nil && s.sender != nil }

func (s *AuthService) Register(ctx context.Context, email, password string) (db.Account, error) {
	// Gin validates rune length; bcrypt limits passwords to 72 bytes.
	// Enforce this again for non-HTTP callers before hashing or DB writes.
	if !validation.PasswordLength(password) {
		return db.Account{}, auth_model.ErrInvalidPassword
	}
	hash, err := authcrypto.HashPassword(password)
	if err != nil {
		return db.Account{}, err
	}
	account, err := account_model.Register(ctx, account_model.RegisterParams{
		Email:        strings.ToLower(strings.TrimSpace(email)),
		PasswordHash: hash,
	})
	if err != nil {
		return db.Account{}, err
	}
	return account, nil
}

// ResendVerificationEmail is deliberately non-enumerating: unknown or
// already-verified addresses get the same response as a queued message.
// Delivery uses the existing durable outbox and email verification job.
func (s *AuthService) ResendVerificationEmail(ctx context.Context, email string) error {
	q, err := db.GetQueries(ctx)
	if err != nil {
		return err
	}
	account, err := account_model.ReadByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if account.IsEmailVerified {
		return nil
	}
	return q.RecordEmailVerificationJob(ctx, account.ID)
}

type LoginResult struct {
	Account          db.Account
	Session          db.Session
	AccessToken      string
	AccessExpiresAt  time.Time
	RefreshToken     string
	RefreshExpiresAt time.Time
}

func (s *AuthService) Login(ctx context.Context, email, password, userAgent, clientIP string) (LoginResult, error) {
	q, err := db.GetQueries(ctx)
	if err != nil {
		return LoginResult{}, err
	}
	var result LoginResult
	account, err := account_model.ReadByEmail(ctx, strings.TrimSpace(email))
	if errors.Is(err, pgx.ErrNoRows) {
		checkUnknownPassword(password)
		s.recordLoginRejected(ctx, audit_model.LoginRejected{AccountID: 0, Method: audit_model.MethodPassword, Reason: audit_model.ReasonUnknownAccount, Identifier: email, IP: clientIP})
		return result, auth_model.ErrInvalidCredentials
	}
	if err != nil {
		return result, err
	}
	hash, err := q.GetPasswordHash(ctx, account.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		checkUnknownPassword(password)
		s.recordLoginRejected(ctx, audit_model.LoginRejected{AccountID: account.ID, Method: audit_model.MethodPassword, Reason: audit_model.ReasonPasswordUnset, Identifier: email, IP: clientIP})
		return result, auth_model.ErrInvalidCredentials
	}
	if err != nil {
		return result, err
	}
	if err := authcrypto.CheckPassword(password, hash); err != nil {
		s.recordLoginRejected(ctx, audit_model.LoginRejected{AccountID: account.ID, Method: audit_model.MethodPassword, Reason: audit_model.ReasonInvalidPassword, Identifier: email, IP: clientIP})
		return result, auth_model.ErrInvalidCredentials
	}
	return s.issueSession(ctx, account, userAgent, clientIP, hash)
}

func (s *AuthService) VerifyEmail(ctx context.Context, emailID int64, secretCode string) error {
	_, err := account_model.VerifyEmail(ctx, db.ConsumeVerifyEmailParams{
		ID: emailID, SecretCode: secretCode,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return auth_model.ErrInvalidVerificationLink
	}
	return err
}

type RenewResult struct {
	AccessToken          string
	AccessTokenExpiresAt time.Time
}

func (s *AuthService) Renew(ctx context.Context, refreshToken string) (RenewResult, error) {
	q, err := db.GetQueries(ctx)
	if err != nil {
		return RenewResult{}, err
	}
	var result RenewResult
	payload, err := s.maker.VerifyToken(refreshToken, authcrypto.TokenTypeRefresh)
	if err != nil {
		return result, auth_model.ErrInvalidRefreshToken
	}
	session, err := q.GetSession(ctx, payload.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		return result, auth_model.ErrSessionUnavailable
	}
	if err != nil {
		return result, err
	}
	if !auth_model.ValidRefreshSession(session, payload.AccountID, refreshToken, time.Now()) {
		return result, auth_model.ErrSessionUnavailable
	}
	access, tokenPayload, err := s.maker.CreateSessionAccessToken(
		payload.AccountID, s.config.Authentication().AccessTokenTTL, session.ID,
	)
	if err != nil {
		return result, err
	}
	return RenewResult{AccessToken: access, AccessTokenExpiresAt: tokenPayload.ExpiredAt}, nil
}

func (s *AuthService) Revoke(ctx context.Context, refreshToken string) error {
	q, err := db.GetQueries(ctx)
	if err != nil {
		return err
	}
	payload, err := s.maker.VerifyToken(refreshToken, authcrypto.TokenTypeRefresh)
	if err != nil {
		return auth_model.ErrInvalidRefreshToken
	}
	params := db.BlockSessionParams{
		ID: payload.ID, AccountID: payload.AccountID,
		RefreshToken: auth_model.RefreshTokenDigest(refreshToken),
	}
	_, err = q.BlockSession(ctx, params)
	if errors.Is(err, pgx.ErrNoRows) {
		// Transition compatibility: sessions written before digest storage
		// contain the raw JWT. Remove this fallback after their maximum TTL.
		params.RefreshToken = refreshToken
		_, err = q.BlockSession(ctx, params)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return auth_model.ErrSessionUnavailable
	}
	return err
}

// issueSession is the single issuance point shared by password, OIDC and phone logins.
// Every caller must authenticate its proof before reaching this operation.
func (s *AuthService) issueSession(ctx context.Context, account db.Account, userAgent, clientIP string, expectedPasswordHash ...string) (LoginResult, error) {
	var result LoginResult
	auth := s.config.Authentication()
	refreshToken, refreshPayload, err := s.maker.CreateToken(account.ID, auth.RefreshTokenTTL, authcrypto.TokenTypeRefresh)
	if err != nil {
		return result, err
	}
	accessToken, accessPayload, err := s.maker.CreateSessionAccessToken(account.ID, auth.AccessTokenTTL, refreshPayload.ID)
	if err != nil {
		return result, err
	}
	session, err := auth_model.NewSession(ctx, db.CreateSessionParams{
		ID: refreshPayload.ID, AccountID: account.ID, RefreshToken: auth_model.RefreshTokenDigest(refreshToken),
		UserAgent: boundedConnectionUserAgent(userAgent), ClientIp: clientIP, ExpiresAt: refreshPayload.ExpiredAt,
	}, expectedPasswordHash...)
	if err != nil {
		return result, err
	}
	return LoginResult{
		Account: account, Session: session,
		AccessToken: accessToken, AccessExpiresAt: accessPayload.ExpiredAt,
		RefreshToken: refreshToken, RefreshExpiresAt: refreshPayload.ExpiredAt,
	}, nil
}

// RequestPasswordReset only persists a delivery intent. It never performs
// synchronous SMTP or reveals whether an email is registered. Workers create
// time-bound proofs after the durable job reaches Asynq.
func (s *AuthService) RequestPasswordReset(ctx context.Context, email string) error {
	q, err := db.GetQueries(ctx)
	if err != nil {
		return err
	}
	account, err := account_model.ReadByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if !account.IsEmailVerified {
		return nil
	}
	return q.RecordPasswordResetJob(ctx, account.ID)
}

// ResetPassword consumes its token transactionally and revokes refresh sessions.
// Recovery also revokes delegated OAuth authorizations and existing OAuth
// bearer credentials. Old access JWTs remain cryptographically valid but
// newly admitted protected API requests reject their deleted sessions.
func (s *AuthService) ResetPassword(ctx context.Context, secret, password string) error {
	if !validation.PasswordLength(password) {
		return auth_model.ErrInvalidPassword
	}
	if len(secret) != 43 {
		return auth_model.ErrInvalidResetToken
	}
	digest := sha256.Sum256([]byte(secret))
	hash, err := authcrypto.HashPassword(password)
	if err != nil {
		return err
	}
	err = auth_model.ResetPassword(ctx, digest[:], hash)
	if errors.Is(err, auth_model.ErrPasswordResetInvalid) {
		return auth_model.ErrInvalidResetToken
	}
	return err
}

func (s *AuthService) StartEmailLink(ctx context.Context, accountID int64, email, phoneCode string) error {
	if s.emailSender == nil {
		return fmt.Errorf("email link mailer unavailable")
	}
	email, valid := validation.EmailAddress(email)
	if !valid {
		return auth_model.ErrInvalidEmail
	}
	account, err := account_model.ReadByID(ctx, accountID)
	if err != nil {
		return err
	}
	if account.Email != "" {
		return auth_model.ErrEmailAlreadyLinked
	}
	_, err = account_model.ReadByEmail(ctx, email)
	if err == nil {
		return account_model.ErrEmailRegistered
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	if err := s.AuthorizePhoneProof(ctx, accountID, phoneCode); err != nil {
		return err
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return err
	}
	secret := base64.RawURLEncoding.EncodeToString(raw)
	hash := sha256.Sum256([]byte(secret))
	if err := account_model.BeginEmailLink(ctx, accountID, email, hash[:]); err != nil {
		switch {
		case errors.Is(err, account_model.ErrEmailLinkTaken):
			return account_model.ErrEmailRegistered
		case errors.Is(err, account_model.ErrEmailLinkUnavailable):
			return auth_model.ErrEmailAlreadyLinked
		case errors.Is(err, account_model.ErrEmailLinkCooldown):
			return auth_model.ErrEmailLinkCooldown
		}
		return err
	}
	content := fmt.Sprintf("To link this email to your BDSPro account, sign in and submit this one-time token to POST /users/me/email/verify within 15 minutes: <code>%s</code>", html.EscapeString(secret))
	return s.emailSender.SendEmail("Verify your BDSPro email link", content, []string{email}, nil, nil, nil)
}

func (s *AuthService) ConfirmEmailLink(ctx context.Context, accountID int64, secret string) error {
	if len(secret) != 43 {
		return auth_model.ErrInvalidEmailLink
	}
	hash := sha256.Sum256([]byte(secret))
	err := account_model.ConfirmEmailLink(ctx, accountID, hash[:])
	if errors.Is(err, account_model.ErrEmailLinkInvalid) {
		return auth_model.ErrInvalidEmailLink
	}
	if errors.Is(err, account_model.ErrEmailLinkUnavailable) {
		return auth_model.ErrEmailAlreadyLinked
	}
	if errors.Is(err, account_model.ErrEmailLinkTaken) {
		return account_model.ErrEmailRegistered
	}
	return err
}

var phoneE164 = regexp.MustCompile(`^\+[1-9][0-9]{7,14}$`)
var phoneCode = regexp.MustCompile(`^[0-9]{6}$`)

// Phone challenge verification is a method of the canonical AuthService.
func (s *AuthService) phoneAllowed() error {
	if s.config.Environment != "development" && s.sender == nil {
		return errors.New("phone OTP delivery unavailable")
	}
	return nil
}
func (s *AuthService) generatePhoneCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}
func (s *AuthService) phoneCodeMAC(phone, purpose string, accountID int64, code string) []byte {
	h := hmac.New(sha256.New, []byte(s.config.TokenSymmetricKey))
	_, _ = fmt.Fprintf(h, "bdspro:phone-otp:v1|%s|%s|%d|%s", purpose, phone, accountID, code)
	return h.Sum(nil)
}
func (s *AuthService) startPhoneChallenge(ctx context.Context, phone, purpose string, accountID int64) error {
	code, err := s.generatePhoneCode()
	if err != nil {
		return err
	}
	devCode := ""
	if s.config.Environment == "development" {
		devCode = code
	}
	if err := auth_model.StartChallenge(ctx, phone, purpose, accountID, s.phoneCodeMAC(phone, purpose, accountID, code), devCode); err != nil {
		return err
	}
	if s.sender == nil {
		return nil // Only a disposable development database permits this.
	}
	return s.sender.SendCode(ctx, phone, code)
}

// StartBind authenticates an account with its existing BDSPro password.
// The challenge can only be created after fresh proof of account control.
func (s *AuthService) StartPhoneBind(ctx context.Context, accountID int64, password, phone string) error {
	q, err := db.GetQueries(ctx)
	if err != nil {
		return err
	}
	if err := s.phoneAllowed(); err != nil {
		return err
	}
	if !phoneE164.MatchString(phone) {
		return auth_model.ErrInvalidPhone
	}
	hash, err := q.GetPasswordHash(ctx, accountID)
	if errors.Is(err, pgx.ErrNoRows) {
		return auth_model.ErrInvalidCredentials
	}
	if err != nil {
		return err
	}
	if authcrypto.CheckPassword(password, hash) != nil {
		return auth_model.ErrInvalidCredentials
	}
	return s.startPhoneBind(ctx, accountID, phone)
}

// StartOAuth2Bind requires a fresh OIDC assertion whose issuer/subject is
// already linked to this very account. Matching email never authorizes binding.
func (s *AuthService) StartOAuth2PhoneBind(ctx context.Context, accountID int64, identity auth_model.OIDCClaims, phone string) error {
	q, err := db.GetQueries(ctx)
	if err != nil {
		return err
	}
	if err := s.phoneAllowed(); err != nil {
		return err
	}
	if !phoneE164.MatchString(phone) {
		return auth_model.ErrInvalidPhone
	}
	key, err := identity.Key()
	if err != nil || !s.config.OIDC.Enabled() || key.Issuer != s.config.OIDC.Issuer {
		return auth_model.ErrInvalidCredentials
	}
	linkedID, err := auth_model.GetOAuth2AccountID(ctx, q, key)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && linkedID != accountID) {
		return auth_model.ErrInvalidCredentials
	}
	if err != nil {
		return err
	}
	return s.startPhoneBind(ctx, accountID, phone)
}

func (s *AuthService) startPhoneBind(ctx context.Context, accountID int64, phone string) error {
	q, err := db.GetQueries(ctx)
	if err != nil {
		return err
	}
	if _, err := account_model.ReadByID(ctx, accountID); err != nil {
		return err
	}
	// A previously verified account cannot start binding another number.
	// The unique account_id constraint remains the concurrency safeguard.
	if _, lookupErr := q.GetPhoneForAccount(ctx, accountID); lookupErr == nil {
		return auth_model.ErrPhoneUnavailable
	} else if !errors.Is(lookupErr, pgx.ErrNoRows) {
		return lookupErr
	}
	existing, err := auth_model.AccountIDForPhone(ctx, phone)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	// Bind is for the first verified number only. Do not reuse it for a
	// phone-change workflow, even when the number already belongs to us.
	if err == nil || existing != 0 {
		return auth_model.ErrPhoneUnavailable
	}
	err = s.startPhoneChallenge(ctx, phone, "bind", accountID)
	return err
}
func (s *AuthService) VerifyPhoneBind(ctx context.Context, accountID int64, phone, code string) error {
	if err := s.phoneAllowed(); err != nil {
		return err
	}
	if !phoneE164.MatchString(phone) || !phoneCode.MatchString(code) {
		return auth_model.ErrOTPInvalid
	}
	return auth_model.ConsumeChallenge(ctx, phone, "bind", accountID, s.phoneCodeMAC(phone, "bind", accountID, code))
}

// StartRegister sends an OTP before any account is created. A registered
// phone returns a distinct conflict, while signup remains unique in PostgreSQL.
func (s *AuthService) StartPhoneRegister(ctx context.Context, phone string) error {
	if err := s.phoneAllowed(); err != nil {
		return err
	}
	if !phoneE164.MatchString(phone) {
		return auth_model.ErrInvalidPhone
	}
	_, err := auth_model.AccountIDForPhone(ctx, phone)
	if err == nil {
		return auth_model.ErrPhoneRegistered
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	code, err := s.generatePhoneCode()
	if err != nil {
		return err
	}
	devCode := ""
	if s.config.Environment == "development" {
		devCode = code
	}
	if err = auth_model.StartPhoneSignup(ctx, phone, s.phoneCodeMAC(phone, "register", 0, code), devCode); err != nil {
		if errors.Is(err, auth_model.ErrOTPCooldown) {
			return auth_model.ErrOTPCooldown
		}
		return err
	}
	if s.sender == nil {
		return nil
	}
	return s.sender.SendCode(ctx, phone, code)
}

// VerifyRegister creates a phone-only Account and issues the same session
// contract used by password and OIDC sign-in. A failed session issuance
// leaves the verified account accessible through OTP login.
func (s *AuthService) VerifyPhoneRegister(ctx context.Context, phone, code, userAgent, clientIP string) (LoginResult, error) {
	var empty LoginResult
	if err := s.phoneAllowed(); err != nil {
		return empty, err
	}
	if !phoneE164.MatchString(phone) || !phoneCode.MatchString(code) {
		return empty, auth_model.ErrOTPInvalid
	}
	a, err := auth_model.CompletePhoneSignup(ctx, phone, s.phoneCodeMAC(phone, "register", 0, code))
	if err != nil {
		return empty, err
	}
	return s.issueSession(ctx, a, userAgent, clientIP)
}

// StartLogin does not reveal whether a number exists. Unknown numbers
// receive the same accepted response, but no OTP is delivered.
func (s *AuthService) StartPhoneLogin(ctx context.Context, phone string) error {
	if err := s.phoneAllowed(); err != nil {
		return err
	}
	if !phoneE164.MatchString(phone) {
		return auth_model.ErrInvalidPhone
	}
	accountID, err := auth_model.AccountIDForPhone(ctx, phone)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	err = s.startPhoneChallenge(ctx, phone, "login", accountID)
	if errors.Is(err, auth_model.ErrOTPCooldown) {
		// An existing number and an unknown number must not be distinguishable.
		return nil
	}
	return err
}
func (s *AuthService) VerifyPhoneLogin(ctx context.Context, phone, code, userAgent, clientIP string) (LoginResult, error) {
	var empty LoginResult
	if err := s.phoneAllowed(); err != nil {
		return empty, err
	}
	if !phoneE164.MatchString(phone) || !phoneCode.MatchString(code) {
		if phoneE164.MatchString(phone) {
			s.recordLoginRejected(ctx, audit_model.LoginRejected{AccountID: 0, Method: audit_model.MethodPhoneOTP, Reason: audit_model.ReasonInvalidOTP, Identifier: phone, IP: clientIP})
		}
		return empty, auth_model.ErrOTPInvalid
	}
	accountID, err := auth_model.AccountIDForPhone(ctx, phone)
	if errors.Is(err, pgx.ErrNoRows) {
		s.recordLoginRejected(ctx, audit_model.LoginRejected{AccountID: 0, Method: audit_model.MethodPhoneOTP, Reason: audit_model.ReasonUnknownAccount, Identifier: phone, IP: clientIP})
		return empty, auth_model.ErrOTPInvalid
	}
	if err != nil {
		return empty, err
	}
	if err := auth_model.ConsumeChallenge(ctx, phone, "login", accountID, s.phoneCodeMAC(phone, "login", accountID, code)); err != nil {
		if errors.Is(err, auth_model.ErrOTPInvalid) {
			s.recordLoginRejected(ctx, audit_model.LoginRejected{AccountID: accountID, Method: audit_model.MethodPhoneOTP, Reason: audit_model.ReasonInvalidOTP, Identifier: phone, IP: clientIP})
			return empty, auth_model.ErrOTPInvalid
		}
		return empty, err
	}
	account, err := account_model.ReadByID(ctx, accountID)
	if err != nil {
		return empty, err
	}
	return s.issueSession(ctx, account, userAgent, clientIP)
}

// AuthorizePhoneProof proves fresh possession of the *already bound* phone.
// The login OTP is single-use, so it cannot be replayed for OAuth linking.
// A linked OAuth2 credential is not inferred from a matching email address.
func (s *AuthService) AuthorizePhoneProof(ctx context.Context, accountID int64, code string) error {
	q, err := db.GetQueries(ctx)
	if err != nil {
		return err
	}
	if !phoneCode.MatchString(code) {
		return auth_model.ErrOTPInvalid
	}
	phone, err := q.GetPhoneForAccount(ctx, accountID)
	if errors.Is(err, pgx.ErrNoRows) {
		return auth_model.ErrInvalidCredentials
	}
	if err != nil {
		return err
	}
	err = auth_model.ConsumeChallenge(ctx, phone, "login", accountID, s.phoneCodeMAC(phone, "login", accountID, code))
	if errors.Is(err, auth_model.ErrOTPInvalid) {
		return auth_model.ErrOTPInvalid
	}
	return err
}

// CurrentAccount reads persisted identity instead of echoing a JWT claim.
// All credentials converge on the same account representation.
func (s *AuthService) CurrentAccount(ctx context.Context, id int64) (db.Account, error) {
	return account_model.GetByID(ctx, id)
}

// A fixed bcrypt hash at the same cost as freshly issued password hashes.
// It belongs to no account and is deliberately not a credential or a secret.
// Missing-account and passwordless-account login attempts must still pay
// the normal bcrypt verification cost before returning InvalidCredentials.
const dummyPasswordHash = "$2a$10$jf3HO8nLz6D6fJUAiJ7sB.wHyxxdZneCFqfIxSyWP4t9wbygi36Ya"

func checkUnknownPassword(password string) {
	_ = authcrypto.CheckPassword(password, dummyPasswordHash)
}

// GetConnections reads the authenticated Account's latest connection events.
// The HTTP boundary supplies accountID from verified JWT claims, not request input.
func (s *AuthService) GetConnections(ctx context.Context, accountID int64) ([]db.ListAccountConnectionHistoriesRow, error) {
	return account_model.GetConnections(ctx, accountID)
}

// boundedConnectionUserAgent preserves valid Unicode while respecting the
// connection-history column's 2048-character constraint. Invalid UTF-8 and
// PostgreSQL-disallowed NUL bytes must not turn valid login proofs into 500s.
func boundedConnectionUserAgent(value string) string {
	const maxRunes = 2048
	value = strings.ToValidUTF8(strings.ReplaceAll(value, "\x00", ""), "")
	if utf8.RuneCountInString(value) <= maxRunes {
		return value
	}
	return string([]rune(value)[:maxRunes])
}

// recordLoginRejected is best-effort for a rejected proof: inability
// to record an unsuccessful login must not convert a 401 into a 5xx or allow
// a credential to be issued. Successful session-history writes remain atomic.
func (s *AuthService) recordLoginRejected(ctx context.Context, event audit_model.LoginRejected) {
	if err := audit_model.RecordLoginRejected(ctx, s.config.AuditKey, event); err != nil {
		log.Ctx(ctx).Warn().
			Str("method", event.Method).
			Str("error_type", fmt.Sprintf("%T", err)).
			Msg("audit.login_record_failed")
	}
}
