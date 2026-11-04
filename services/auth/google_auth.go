package auth

import (
	"context"
	"errors"
	auth_model "github.com/datvtph41107/bdspro/models/auth"
	"github.com/jackc/pgx/v5/pgconn"
	"strings"

	account_model "github.com/datvtph41107/bdspro/models/account"
	db "github.com/datvtph41107/bdspro/models/db"
	authcrypto "github.com/datvtph41107/bdspro/modules/auth"
	"github.com/datvtph41107/bdspro/modules/setting"
	"github.com/datvtph41107/bdspro/modules/token"
	"github.com/jackc/pgx/v5"
)

type GoogleAuthService struct {
	store  db.Store
	maker  token.Maker
	config setting.Config
}

func NewGoogleAuthService(store db.Store, maker token.Maker, cfg setting.Config) *GoogleAuthService {
	return &GoogleAuthService{store: store, maker: maker, config: cfg}
}

// CheckPassword requires an existing password credential to link Google.
// Passwordless accounts need a separate reauthentication method.
func (s *GoogleAuthService) CheckPassword(ctx context.Context, accountID int64, password string) error {
	_, err := s.store.GetAccountByID(ctx, accountID)
	if errors.Is(err, pgx.ErrNoRows) {
		return auth_model.ErrInvalidCredentials
	}
	if err != nil {
		return err
	}
	hash, err := s.store.GetPasswordHash(ctx, accountID)
	if errors.Is(err, pgx.ErrNoRows) {
		return auth_model.ErrInvalidCredentials
	}
	if err != nil {
		return err
	}
	if authcrypto.CheckPassword(password, hash) != nil {
		return auth_model.ErrInvalidCredentials
	}
	return nil
}

func (s *GoogleAuthService) Link(ctx context.Context, accountID int64, subject string) error {
	err := s.store.LinkGoogleIdentity(ctx, db.LinkGoogleIdentityParams{
		AccountID: accountID, GoogleSub: subject,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return auth_model.ErrGoogleIdentityAlreadyLinked
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return errors.Join(auth_model.ErrGoogleAccountAlreadyLinked, err)
	}
	return err
}

// Login logs in a linked Google subject, or registers a new account when the
// signed ID token provides a verified email. Email matching never links accounts.
func (s *GoogleAuthService) Login(
	ctx context.Context, identity VerifiedGoogleIdentity, userAgent, clientIP string,
) (LoginResult, error) {
	var empty LoginResult
	if identity.Subject == "" {
		return empty, auth_model.ErrGoogleIdentityUnlinked
	}
	accountID, err := s.store.GetGoogleAccountID(ctx, identity.Subject)
	var account db.Account
	if errors.Is(err, pgx.ErrNoRows) {
		email := strings.ToLower(strings.TrimSpace(identity.Email))
		if !identity.EmailVerified || email == "" {
			return empty, auth_model.ErrGoogleEmailUnverified
		}
		account, err = account_model.CreateGoogleAccount(ctx, s.store, email, identity.Subject)
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "accounts_email_login_uq" {
				return empty, errors.Join(auth_model.ErrGoogleEmailAlreadyRegistered, err)
			}
			// Concurrent first login: the other transaction may have linked
			// the same subject. Only resolve by sub, never by email.
			linkedID, lookupErr := s.store.GetGoogleAccountID(ctx, identity.Subject)
			if lookupErr != nil {
				return empty, err
			}
			account, err = s.store.GetAccountByID(ctx, linkedID)
		}
	} else if err == nil {
		account, err = s.store.GetAccountByID(ctx, accountID)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return empty, account_model.ErrAccountUnavailable
	}
	if err != nil {
		return empty, err
	}
	return IssueSession(ctx, s.store, s.maker, s.config, account, userAgent, clientIP)
}
