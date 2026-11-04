package auth

import (
	"context"
	"errors"
	"strings"

	account_model "github.com/datvtph41107/bdspro/models/account"
	auth_model "github.com/datvtph41107/bdspro/models/auth"
	db "github.com/datvtph41107/bdspro/models/db"
	authcrypto "github.com/datvtph41107/bdspro/modules/auth"
	"github.com/datvtph41107/bdspro/modules/setting"
	"github.com/datvtph41107/bdspro/modules/token"
	"github.com/jackc/pgx/v5"
)

// OAuth2AuthService orchestrates account and session operations from
// cryptographically verified OpenID Connect identity assertions.
type OAuth2AuthService struct {
	store db.Store
	maker token.Maker
	cfg   setting.Config
}

func NewOAuth2AuthService(store db.Store, maker token.Maker, cfg setting.Config) *OAuth2AuthService {
	return &OAuth2AuthService{store: store, maker: maker, cfg: cfg}
}

func (s *OAuth2AuthService) identityKey(identity auth_model.OIDCIdentity) (auth_model.IdentityKey, error) {
	key, err := identity.Key()
	if err != nil {
		return key, err
	}
	if !s.cfg.OIDC.Enabled() || key.Issuer != s.cfg.OIDC.Issuer {
		return auth_model.IdentityKey{}, auth_model.ErrOIDCIssuerInvalid
	}
	return key, nil
}

// CheckPassword authenticates the local account before starting a link.
// A verified external email address cannot authorize linking by itself.
func (s *OAuth2AuthService) CheckPassword(ctx context.Context, accountID int64, password string) error {
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

func (s *OAuth2AuthService) Link(ctx context.Context, accountID int64, identity auth_model.OIDCIdentity) error {
	if accountID <= 0 {
		return auth_model.ErrInvalidCredentials
	}
	key, err := s.identityKey(identity)
	if err != nil {
		return err
	}
	return auth_model.LinkOAuth2Identity(ctx, s.store, accountID, key)
}

// Login never automatically links identities to an existing email account.
// On concurrent first logins it recovers only by the exact issuer/subject key.
func (s *OAuth2AuthService) Login(ctx context.Context, identity auth_model.OIDCIdentity, userAgent, clientIP string) (LoginResult, error) {
	var empty LoginResult
	key, err := s.identityKey(identity)
	if err != nil {
		return empty, err
	}
	accountID, err := auth_model.GetOAuth2AccountID(ctx, s.store, key)
	var account db.Account
	switch {
	case err == nil:
		account, err = s.store.GetAccountByID(ctx, accountID)
	case errors.Is(err, pgx.ErrNoRows):
		if !identity.EmailVerified || strings.TrimSpace(identity.Email) == "" {
			return empty, auth_model.ErrOAuth2EmailUnverified
		}
		account, err = account_model.CreateOAuth2Account(ctx, s.store, identity)
		if err != nil {
			linkedID, lookupErr := auth_model.GetOAuth2AccountID(ctx, s.store, key)
			if lookupErr == nil {
				account, err = s.store.GetAccountByID(ctx, linkedID)
			} else if _, emailErr := s.store.GetAccountByEmail(ctx, identity.Email); emailErr == nil {
				return empty, auth_model.ErrOAuth2EmailAlreadyRegistered
			}
		}
	default:
		return empty, err
	}
	if err != nil {
		return empty, err
	}
	return IssueSession(ctx, s.store, s.maker, s.cfg, account, userAgent, clientIP)
}
