package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"html"
	netmail "net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/datvtph41107/bdspro/internal/config"
	db "github.com/datvtph41107/bdspro/internal/database"
	apperrors "github.com/datvtph41107/bdspro/internal/errors"
	"github.com/datvtph41107/bdspro/internal/mail"
	"github.com/datvtph41107/bdspro/internal/token"
	"github.com/datvtph41107/bdspro/internal/util"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

// AuthService owns registration, credential authentication, email verification,
// access renewal and session revocation. No Gin or HTTP response types enter
// these operations.
type AuthService struct {
	store  db.Store
	maker  token.Maker
	config config.Config
}

func NewAuthService(store db.Store, maker token.Maker, cfg config.Config) *AuthService {
	return &AuthService{store: store, maker: maker, config: cfg}
}

func (s *AuthService) Register(ctx context.Context, email, password string) (db.Account, error) {
	// Gin validates rune length; bcrypt limits passwords to 72 bytes.
	// Enforce this again for non-HTTP callers before hashing or DB writes.
	if utf8.RuneCountInString(password) < 8 || len(password) > 72 {
		return db.Account{}, apperrors.InvalidPassword
	}
	hash, err := util.HashPassword(password)
	if err != nil {
		return db.Account{}, err
	}
	account, err := s.store.RegisterTx(ctx, db.RegisterTxParams{
		Email:        strings.ToLower(strings.TrimSpace(email)),
		PasswordHash: hash,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "accounts_email_login_uq" {
			return db.Account{}, apperrors.Wrap(apperrors.EmailRegistered, err)
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

// Login persists password authentication outcomes before returning to HTTP.
// On audit failure, credentials are not returned to the caller.
func (s *AuthService) Login(ctx context.Context, email, password, userAgent, clientIP string) (LoginResult, error) {
	var result LoginResult
	account, err := s.store.GetAccountByEmail(ctx, strings.TrimSpace(email))
	if errors.Is(err, pgx.ErrNoRows) {
		if err := s.recordPasswordLoginEvent(ctx, 0, "rejected", userAgent, clientIP); err != nil {
			return result, err
		}
		return result, apperrors.InvalidCredentials
	}
	if err != nil {
		return result, err
	}
	hash, err := s.store.GetPasswordHash(ctx, account.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		if err := s.recordPasswordLoginEvent(ctx, account.ID, "rejected", userAgent, clientIP); err != nil {
			return result, err
		}
		return result, apperrors.InvalidCredentials
	}
	if err != nil {
		return result, err
	}
	if err := util.CheckPassword(password, hash); err != nil {
		if err := s.recordPasswordLoginEvent(ctx, account.ID, "rejected", userAgent, clientIP); err != nil {
			return result, err
		}
		return result, apperrors.InvalidCredentials
	}
	issued, err := IssueSession(ctx, s.store, s.maker, s.config, account, userAgent, clientIP)
	if err != nil {
		return result, err
	}
	if err := s.recordPasswordLoginEvent(ctx, account.ID, "succeeded", userAgent, clientIP); err != nil {
		// Credentials are not returned when the security event cannot be
		// persisted. Attempt to revoke the newly created refresh session.
		_, revokeErr := s.store.BlockSession(ctx, db.BlockSessionParams{
			ID: issued.Session.ID, AccountID: account.ID, RefreshToken: issued.RefreshToken,
		})
		return result, errors.Join(err, revokeErr)
	}
	return issued, nil
}

func (s *AuthService) recordPasswordLoginEvent(ctx context.Context, accountID int64, outcome, userAgent, clientIP string) error {
	account := pgtype.Int8{}
	if accountID > 0 {
		account = pgtype.Int8{Int64: accountID, Valid: true}
	}
	return s.store.RecordPasswordLoginEvent(ctx, db.RecordPasswordLoginEventParams{
		AccountID: account,
		Outcome:   outcome,
		UserAgent: truncateLoginMetadata(userAgent, 512),
		ClientIp:  truncateLoginMetadata(clientIP, 64),
	})
}

func truncateLoginMetadata(value string, limit int) string {
	runes := []rune(value)
	if len(runes) > limit {
		return string(runes[:limit])
	}
	return value
}

func (s *AuthService) LoginHistory(ctx context.Context, accountID int64) ([]db.ListOwnLoginEventsRow, error) {
	return s.store.ListOwnLoginEvents(ctx, pgtype.Int8{Int64: accountID, Valid: true})
}

func (s *AuthService) VerifyEmail(ctx context.Context, emailID int64, secretCode string) error {
	_, err := s.store.VerifyEmailTx(ctx, db.ConsumeVerifyEmailParams{
		ID: emailID, SecretCode: secretCode,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return apperrors.InvalidVerificationLink
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
		return result, apperrors.InvalidRefreshToken
	}
	session, err := s.store.GetSession(ctx, payload.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		return result, apperrors.SessionUnavailable
	}
	if err != nil {
		return result, err
	}
	if session.IsBlocked || session.AccountID != payload.AccountID ||
		session.RefreshToken != refreshToken || !time.Now().Before(session.ExpiresAt) {
		return result, apperrors.SessionUnavailable
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
		return apperrors.InvalidRefreshToken
	}
	_, err = s.store.BlockSession(ctx, db.BlockSessionParams{
		ID: payload.ID, AccountID: payload.AccountID, RefreshToken: refreshToken,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return apperrors.SessionUnavailable
	}
	return err
}

// IssueSession is shared by password, Google and phone login, after the
// corresponding identity provider has already authenticated the account.
func IssueSession(ctx context.Context, store db.Store, maker token.Maker, cfg config.Config, account db.Account, userAgent, clientIP string) (LoginResult, error) {
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
	session, err := store.CreateSession(ctx, db.CreateSessionParams{
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
	if utf8.RuneCountInString(password) < 8 || len(password) > 72 {
		return apperrors.InvalidPassword
	}
	if len(secret) != 43 {
		return apperrors.InvalidResetToken
	}
	digest := sha256.Sum256([]byte(secret))
	hash, err := util.HashPassword(password)
	if err != nil {
		return err
	}
	err = s.store.ResetPasswordTx(ctx, digest[:], hash)
	if errors.Is(err, db.ErrPasswordResetInvalid) {
		return apperrors.InvalidResetToken
	}
	return err
}

func (s *AuthService) StartEmailLink(ctx context.Context, accountID int64, email, phoneCode string, sender mail.EmailSender) error {
	if sender == nil {
		return fmt.Errorf("email link mailer unavailable")
	}
	email = strings.ToLower(strings.TrimSpace(email))
	address, err := netmail.ParseAddress(email)
	if err != nil || address.Address != email || len(email) > 254 {
		return apperrors.InvalidEmail
	}
	account, err := s.store.GetAccountByID(ctx, accountID)
	if err != nil {
		return err
	}
	if account.Email != "" {
		return apperrors.EmailAlreadyLinked
	}
	_, err = s.store.GetAccountByEmail(ctx, email)
	if err == nil {
		return apperrors.EmailRegistered
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
	if err := s.store.BeginEmailLinkTx(ctx, accountID, email, hash[:]); err != nil {
		switch {
		case errors.Is(err, db.ErrEmailLinkTaken):
			return apperrors.EmailRegistered
		case errors.Is(err, db.ErrEmailLinkUnavailable):
			return apperrors.EmailAlreadyLinked
		case errors.Is(err, db.ErrEmailLinkCooldown):
			return apperrors.EmailLinkCooldown
		}
		return err
	}
	content := fmt.Sprintf("To link this email to your BDSPro account, sign in and submit this one-time token to POST /users/me/email/verify within 15 minutes: <code>%s</code>", html.EscapeString(secret))
	return sender.SendEmail("Verify your BDSPro email link", content, []string{email}, nil, nil, nil)
}

func (s *AuthService) ConfirmEmailLink(ctx context.Context, accountID int64, secret string) error {
	if len(secret) != 43 {
		return apperrors.InvalidEmailLink
	}
	hash := sha256.Sum256([]byte(secret))
	err := s.store.ConfirmEmailLinkTx(ctx, accountID, hash[:])
	if errors.Is(err, db.ErrEmailLinkInvalid) {
		return apperrors.InvalidEmailLink
	}
	if errors.Is(err, db.ErrEmailLinkUnavailable) {
		return apperrors.EmailAlreadyLinked
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "accounts_email_login_uq" {
		return apperrors.EmailRegistered
	}
	return err
}
