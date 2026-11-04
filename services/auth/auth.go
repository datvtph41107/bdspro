package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	auth_model "github.com/datvtph41107/bdspro/models/auth"
	"github.com/datvtph41107/bdspro/modules/validation"
	"html"
	"strings"
	"time"

	account_model "github.com/datvtph41107/bdspro/models/account"
	db "github.com/datvtph41107/bdspro/models/db"
	authcrypto "github.com/datvtph41107/bdspro/modules/auth"
	"github.com/datvtph41107/bdspro/modules/setting"
	"github.com/datvtph41107/bdspro/modules/token"
	"github.com/datvtph41107/bdspro/services/mail"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// AuthService owns registration, credential authentication, email verification,
// access renewal and session revocation. No Gin or HTTP response types enter
// these operations.
type AuthService struct {
	store  db.Store
	maker  token.Maker
	config setting.Config
}

func NewAuthService(store db.Store, maker token.Maker, cfg setting.Config) *AuthService {
	return &AuthService{store: store, maker: maker, config: cfg}
}

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
	account, err := account_model.Register(ctx, s.store, account_model.RegisterParams{
		Email:        strings.ToLower(strings.TrimSpace(email)),
		PasswordHash: hash,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "accounts_email_login_uq" {
			return db.Account{}, errors.Join(account_model.ErrEmailRegistered, err)
		}
		return db.Account{}, err
	}
	return account, nil
}

// ResendVerificationEmail is deliberately non-enumerating: unknown or
// already-verified addresses get the same response as a queued message.
// Delivery uses the existing durable outbox and email verification job.
func (s *AuthService) ResendVerificationEmail(ctx context.Context, email string) error {
	account, err := s.store.GetAccountByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if account.IsEmailVerified {
		return nil
	}
	return s.store.RecordEmailVerificationJob(ctx, account.ID)
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
	var result LoginResult
	account, err := s.store.GetAccountByEmail(ctx, strings.TrimSpace(email))
	if errors.Is(err, pgx.ErrNoRows) {
		return result, auth_model.ErrInvalidCredentials
	}
	if err != nil {
		return result, err
	}
	hash, err := s.store.GetPasswordHash(ctx, account.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		return result, auth_model.ErrInvalidCredentials
	}
	if err != nil {
		return result, err
	}
	if err := authcrypto.CheckPassword(password, hash); err != nil {
		return result, auth_model.ErrInvalidCredentials
	}
	return IssueSession(ctx, s.store, s.maker, s.config, account, userAgent, clientIP)
}

func (s *AuthService) VerifyEmail(ctx context.Context, emailID int64, secretCode string) error {
	_, err := account_model.VerifyEmail(ctx, s.store, db.ConsumeVerifyEmailParams{
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
	var result RenewResult
	payload, err := s.maker.VerifyToken(refreshToken, token.TokenTypeRefresh)
	if err != nil {
		return result, auth_model.ErrInvalidRefreshToken
	}
	session, err := s.store.GetSession(ctx, payload.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		return result, auth_model.ErrSessionUnavailable
	}
	if err != nil {
		return result, err
	}
	if !auth_model.ValidRefreshSession(session, payload.AccountID, refreshToken, time.Now()) {
		return result, auth_model.ErrSessionUnavailable
	}
	access, tokenPayload, err := s.maker.CreateToken(
		payload.AccountID, s.config.Authentication().AccessTokenTTL, token.TokenTypeAccess,
	)
	if err != nil {
		return result, err
	}
	return RenewResult{AccessToken: access, AccessTokenExpiresAt: tokenPayload.ExpiredAt}, nil
}

func (s *AuthService) Revoke(ctx context.Context, refreshToken string) error {
	payload, err := s.maker.VerifyToken(refreshToken, token.TokenTypeRefresh)
	if err != nil {
		return auth_model.ErrInvalidRefreshToken
	}
	_, err = s.store.BlockSession(ctx, db.BlockSessionParams{
		ID: payload.ID, AccountID: payload.AccountID, RefreshToken: refreshToken,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return auth_model.ErrSessionUnavailable
	}
	return err
}

// IssueSession is shared by password, Google and phone login, after the
// corresponding identity provider has already authenticated the account.
func IssueSession(ctx context.Context, store db.Store, maker token.Maker, cfg setting.Config, account db.Account, userAgent, clientIP string) (LoginResult, error) {
	var result LoginResult
	auth := cfg.Authentication()
	accessToken, accessPayload, err := maker.CreateToken(account.ID, auth.AccessTokenTTL, token.TokenTypeAccess)
	if err != nil {
		return result, err
	}
	refreshToken, refreshPayload, err := maker.CreateToken(account.ID, auth.RefreshTokenTTL, token.TokenTypeRefresh)
	if err != nil {
		return result, err
	}
	session, err := auth_model.NewSession(ctx, store, db.CreateSessionParams{
		ID: refreshPayload.ID, AccountID: account.ID, RefreshToken: refreshToken,
		UserAgent: userAgent, ClientIp: clientIP, ExpiresAt: refreshPayload.ExpiredAt,
	})
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
	account, err := s.store.GetAccountByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if !account.IsEmailVerified {
		return nil
	}
	return s.store.RecordPasswordResetJob(ctx, account.ID)
}

// ResetPassword consumes its token transactionally and revokes refresh sessions.
// Issued access JWTs remain valid until their normal expiry.
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
	err = auth_model.ResetPassword(ctx, s.store, digest[:], hash)
	if errors.Is(err, auth_model.ErrPasswordResetInvalid) {
		return auth_model.ErrInvalidResetToken
	}
	return err
}

func (s *AuthService) StartEmailLink(ctx context.Context, accountID int64, email, phoneCode string, sender mail.EmailSender) error {
	if sender == nil {
		return fmt.Errorf("email link mailer unavailable")
	}
	email, valid := validation.EmailAddress(email)
	if !valid {
		return auth_model.ErrInvalidEmail
	}
	account, err := s.store.GetAccountByID(ctx, accountID)
	if err != nil {
		return err
	}
	if account.Email != "" {
		return auth_model.ErrEmailAlreadyLinked
	}
	_, err = s.store.GetAccountByEmail(ctx, email)
	if err == nil {
		return account_model.ErrEmailRegistered
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	proof := PhoneOTP{Store: s.store, Config: s.config}
	if err := proof.AuthorizePhoneProof(ctx, accountID, phoneCode); err != nil {
		return err
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return err
	}
	secret := base64.RawURLEncoding.EncodeToString(raw)
	hash := sha256.Sum256([]byte(secret))
	if err := account_model.BeginEmailLink(ctx, s.store, accountID, email, hash[:]); err != nil {
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
	return sender.SendEmail("Verify your BDSPro email link", content, []string{email}, nil, nil, nil)
}

func (s *AuthService) ConfirmEmailLink(ctx context.Context, accountID int64, secret string) error {
	if len(secret) != 43 {
		return auth_model.ErrInvalidEmailLink
	}
	hash := sha256.Sum256([]byte(secret))
	err := account_model.ConfirmEmailLink(ctx, s.store, accountID, hash[:])
	if errors.Is(err, account_model.ErrEmailLinkInvalid) {
		return auth_model.ErrInvalidEmailLink
	}
	if errors.Is(err, account_model.ErrEmailLinkUnavailable) {
		return auth_model.ErrEmailAlreadyLinked
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "accounts_email_login_uq" {
		return account_model.ErrEmailRegistered
	}
	return err
}
