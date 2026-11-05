package oauth

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"time"

	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/google/uuid"
)

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
}

func bearerHash(raw string) []byte {
	sum := sha256.Sum256([]byte(raw))
	return sum[:]
}

func validBearer(raw string) bool {
	if len(raw) != 43 {
		return false
	}
	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	return err == nil && len(decoded) == 32 && base64.RawURLEncoding.EncodeToString(decoded) == raw
}

func validVerifier(raw string) bool {
	if len(raw) < 43 || len(raw) > 128 {
		return false
	}
	for _, c := range raw {
		if (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') &&
			(c < '0' || c > '9') && c != '-' && c != '.' && c != '_' && c != '~' {
			return false
		}
	}
	return true
}

func (s *Service) tokenClient(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return ErrInvalidClient
	}
	client, err := s.store.GetOAuthClient(ctx, id)
	if errors.Is(err, db.ErrOAuthClientMissing) {
		return ErrInvalidClient
	}
	if err != nil {
		return err
	}
	// Only public PKCE clients can be registered by the current API.
	// Never accept an existing confidential client without checking its secret.
	if len(client.SecretHash) != 0 {
		return ErrInvalidClient
	}
	return nil
}

func (s *Service) ExchangeCode(ctx context.Context, clientID uuid.UUID,
	code, redirect, verifier string) (TokenResponse, error) {
	if !validBearer(code) || !validVerifier(verifier) {
		return TokenResponse{}, ErrInvalidGrant
	}
	if err := s.tokenClient(ctx, clientID); err != nil {
		return TokenResponse{}, err
	}
	if ValidateRedirectURI(redirect) != nil {
		return TokenResponse{}, ErrInvalidGrant
	}
	sum := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(sum[:])
	access, err := newBearer()
	if err != nil {
		return TokenResponse{}, err
	}
	refresh, err := newBearer()
	if err != nil {
		return TokenResponse{}, err
	}
	family, err := uuid.NewRandom()
	if err != nil {
		return TokenResponse{}, err
	}
	now := time.Now()
	grant, err := s.store.RedeemOAuthAuthCode(ctx, clientID, bearerHash(code),
		redirect, challenge, bearerHash(access), bearerHash(refresh),
		now.Add(AccessTokenTTL), now.Add(RefreshTokenTTL), family)
	if errors.Is(err, db.ErrOAuthGrantInvalid) {
		return TokenResponse{}, ErrInvalidGrant
	}
	if err != nil {
		return TokenResponse{}, err
	}
	return TokenResponse{
		AccessToken: access, TokenType: "Bearer",
		ExpiresIn: int64(AccessTokenTTL.Seconds()), RefreshToken: refresh, Scope: grant.Scope,
	}, nil
}

func (s *Service) RefreshToken(ctx context.Context,
	clientID uuid.UUID, refresh string) (TokenResponse, error) {
	if !validBearer(refresh) {
		return TokenResponse{}, ErrInvalidGrant
	}
	if err := s.tokenClient(ctx, clientID); err != nil {
		return TokenResponse{}, err
	}
	access, err := newBearer()
	if err != nil {
		return TokenResponse{}, err
	}
	nextRefresh, err := newBearer()
	if err != nil {
		return TokenResponse{}, err
	}
	grant, err := s.store.RotateOAuthRefreshToken(ctx, clientID,
		bearerHash(refresh), bearerHash(access), bearerHash(nextRefresh),
		time.Now().Add(AccessTokenTTL))
	if errors.Is(err, db.ErrOAuthGrantInvalid) || errors.Is(err, db.ErrOAuthGrantReplay) {
		return TokenResponse{}, ErrInvalidGrant
	}
	if err != nil {
		return TokenResponse{}, err
	}
	return TokenResponse{
		AccessToken: access, TokenType: "Bearer",
		ExpiresIn: int64(AccessTokenTTL.Seconds()), RefreshToken: nextRefresh,
		Scope: grant.Scope,
	}, nil
}

func (s *Service) RevokeToken(ctx context.Context, clientID uuid.UUID, raw string) error {
	if err := s.tokenClient(ctx, clientID); err != nil {
		return err
	}
	// RFC 7009: unknown tokens are not disclosed to the caller.
	if !validBearer(raw) {
		return nil
	}
	return s.store.RevokeOAuthBearer(ctx, clientID, bearerHash(raw))
}

func (s *Service) UserAccess(ctx context.Context, raw string) (db.OAuthAccess, error) {
	if !validBearer(raw) {
		return db.OAuthAccess{}, ErrInvalidGrant
	}
	access, err := s.store.ResolveOAuthAccessToken(ctx, bearerHash(raw))
	if errors.Is(err, db.ErrOAuthGrantInvalid) {
		return db.OAuthAccess{}, ErrInvalidGrant
	}
	return access, err
}
