package service

import (
	"context"
	"errors"

	apperrors "github.com/datvtph41107/bdspro/internal/core/errors"
	"github.com/datvtph41107/bdspro/internal/config"
	db "github.com/datvtph41107/bdspro/internal/db/sqlc"
	"github.com/datvtph41107/bdspro/internal/token"
	"github.com/datvtph41107/bdspro/internal/util"
	"github.com/jackc/pgx/v5"
)

// GoogleAuthService owns account linking and session creation.
// OAuth redirects, state cookies, PKCE, and token exchange stay in the HTTP handler.
type GoogleAuthService struct {
	store db.Store
	maker token.Maker
	config config.Config
}

func NewGoogleAuthService(store db.Store, maker token.Maker, cfg config.Config) *GoogleAuthService {
	return &GoogleAuthService{store: store, maker: maker, config: cfg}
}

func (s *GoogleAuthService) CheckPassword(ctx context.Context, accountID int64, password string) error {
	account, err := s.store.GetAccountByID(ctx, accountID)
	if errors.Is(err, pgx.ErrNoRows) {
		return apperrors.InvalidCredentials
	}
	if err != nil {
		return err
	}
	if util.CheckPassword(password, account.PasswordHash) != nil {
		return apperrors.InvalidCredentials
	}
	return nil
}

func (s *GoogleAuthService) Link(ctx context.Context, accountID int64, subject string) error {
	err := s.store.LinkGoogleIdentity(ctx, db.LinkGoogleIdentityParams{
		AccountID: accountID, GoogleSub: subject,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return apperrors.GoogleIdentityAlreadyLinked
	}
	if pgErr := apperrors.Postgres(err); pgErr != nil && pgErr.Code == "23505" {
		return apperrors.Wrap(apperrors.GoogleAccountAlreadyLinked, err)
	}
	return err
}

func (s *GoogleAuthService) Login(
	ctx context.Context, subject, userAgent, clientIP string,
) (LoginResult, error) {
	var empty LoginResult
	accountID, err := s.store.GetGoogleAccountID(ctx, subject)
	if errors.Is(err, pgx.ErrNoRows) {
		return empty, apperrors.GoogleIdentityUnlinked
	}
	if err != nil {
		return empty, err
	}
	account, err := s.store.GetAccountByID(ctx, accountID)
	if errors.Is(err, pgx.ErrNoRows) {
		return empty, apperrors.AccountUnavailable
	}
	if err != nil {
		return empty, err
	}
	return IssueSession(ctx, s.store, s.maker, s.config, account, userAgent, clientIP)
}
