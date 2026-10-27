package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/datvtph41107/bdspro/config"
	apperrors "github.com/datvtph41107/bdspro/core/errors"
	db "github.com/datvtph41107/bdspro/db/sqlc"
	"github.com/datvtph41107/bdspro/token"
	"github.com/datvtph41107/bdspro/util"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// ErrInvalidCredentials is shared by password and phone authentication.
var ErrInvalidCredentials = apperrors.InvalidCredentials

// AuthService owns registration, credential authentication, email verification,
// access renewal and session revocation. No Gin or HTTP response types enter
// these operations.
type AuthService struct {
	store     db.Store
	scheduler VerificationTaskScheduler
	maker     token.Maker
	config    config.Config
}

func NewAuthService(store db.Store, scheduler VerificationTaskScheduler, maker token.Maker, cfg config.Config) *AuthService {
	return &AuthService{store: store, scheduler: scheduler, maker: maker, config: cfg}
}

func (s *AuthService) Register(ctx context.Context, email, password string) (db.Account, error) {
	hash, err := util.HashPassword(password)
	if err != nil {
		return db.Account{}, err
	}
	account, err := s.store.RegisterTx(ctx, db.RegisterTxParams{
		Email: strings.ToLower(strings.TrimSpace(email)), PasswordHash: hash,
		AfterCreate: func(account db.Account) error {
			return s.scheduler.ScheduleVerifyEmail(ctx, account.ID)
		},
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
		return result, ErrInvalidCredentials
	}
	if err != nil {
		return result, err
	}
	if err := util.CheckPassword(password, account.PasswordHash); err != nil {
		return result, ErrInvalidCredentials
	}
	return IssueSession(ctx, s.store, s.maker, s.config, account, userAgent, clientIP)
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
