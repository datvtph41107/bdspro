package oauth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	auth_model "github.com/datvtph41107/bdspro/models/auth"
	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/google/uuid"
)

const (
	ScopeProfile    = "profile:read"
	AccessTokenTTL  = 15 * time.Minute
	RefreshTokenTTL = 30 * 24 * time.Hour
	AuthCodeTTL     = 5 * time.Minute
)

var (
	ErrInvalidRequest = errors.New("invalid OAuth request")
	ErrInvalidClient  = errors.New("invalid OAuth client")
	ErrInvalidGrant   = errors.New("invalid OAuth grant")
	ErrAccessDenied   = errors.New("OAuth consent denied")
)

type Service struct {
	store db.OAuthServerStore
}

func NewService(store db.OAuthServerStore) *Service {
	return &Service{store: store}
}

type AuthorizationRequest struct {
	ClientID            uuid.UUID
	RedirectURI         string
	ResponseType        string
	CodeChallenge       string
	CodeChallengeMethod string
	Scope               string
	State               string
	ConsentVersion      int64
}

// ParseAuthorizationRequest is the sole route-independent decoding of an OAuth authorization query.
func ParseAuthorizationRequest(values url.Values) (AuthorizationRequest, error) {
	id, err := uuid.Parse(values.Get("client_id"))
	if err != nil {
		return AuthorizationRequest{}, ErrInvalidRequest
	}
	return AuthorizationRequest{
		ClientID: id, RedirectURI: values.Get("redirect_uri"),
		ResponseType: values.Get("response_type"), Scope: values.Get("scope"),
		State: values.Get("state"), CodeChallenge: values.Get("code_challenge"),
		CodeChallengeMethod: values.Get("code_challenge_method"),
	}, nil
}

func ValidateRedirectURI(raw string) error {
	if len(raw) > 2048 || len(raw) == 0 {
		return ErrInvalidRequest
	}
	u, err := url.Parse(raw)
	if err != nil || u == nil || u.Hostname() == "" || u.Opaque != "" ||
		u.Fragment != "" || u.RawFragment != "" || u.User != nil ||
		(u.Scheme != "https" && u.Scheme != "http") || u.String() != raw {
		return ErrInvalidRequest
	}
	// Reject encoded control characters in the registered callback path.
	for _, char := range u.Path {
		if char < 0x20 || char == 0x7f {
			return ErrInvalidRequest
		}
	}
	if u.Scheme == "http" {
		ip := net.ParseIP(u.Hostname())
		if !strings.EqualFold(u.Hostname(), "localhost") && (ip == nil || !ip.IsLoopback()) {
			return ErrInvalidRequest
		}
	}
	return nil
}

func validateAuthorization(a AuthorizationRequest) error {
	if a.ClientID == uuid.Nil || ValidateRedirectURI(a.RedirectURI) != nil ||
		a.ResponseType != "code" || a.Scope != ScopeProfile ||
		a.CodeChallengeMethod != "S256" || len(a.CodeChallenge) != 43 ||
		len(a.State) < 16 || len(a.State) > 256 {
		return ErrInvalidRequest
	}
	b, err := base64.RawURLEncoding.DecodeString(a.CodeChallenge)
	if err != nil || len(b) != sha256.Size ||
		base64.RawURLEncoding.EncodeToString(b) != a.CodeChallenge {
		return ErrInvalidRequest
	}
	for _, ch := range a.State {
		if (ch < 'a' || ch > 'z') && (ch < 'A' || ch > 'Z') &&
			(ch < '0' || ch > '9') && ch != '_' && ch != '-' {
			return ErrInvalidRequest
		}
	}
	return nil
}

func (s *Service) RegisterClient(ctx context.Context, accountID int64, name, redirect string) (*auth_model.OAuthClient, error) {
	name = strings.TrimSpace(name)
	if accountID <= 0 || len(name) == 0 || len(name) > 120 || ValidateRedirectURI(redirect) != nil {
		return nil, ErrInvalidRequest
	}
	id, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}
	row, err := s.store.CreateOAuthClient(ctx, db.OauthClient{
		ID: id, OwnerAccountID: accountID, Name: name, RedirectUri: redirect,
	})
	if err != nil {
		return nil, err
	}
	return auth_model.OAuthClientFromDB(row), nil
}

func (s *Service) ListClients(ctx context.Context, accountID int64) ([]*auth_model.OAuthClient, error) {
	if accountID <= 0 {
		return nil, ErrInvalidRequest
	}
	rows, err := s.store.ListOAuthClients(ctx, accountID)
	if err != nil {
		return nil, err
	}
	clients := make([]*auth_model.OAuthClient, 0, len(rows))
	for _, row := range rows {
		clients = append(clients, auth_model.OAuthClientFromDB(row))
	}
	return clients, nil
}

func (s *Service) DeleteClient(ctx context.Context, accountID int64, id uuid.UUID) error {
	if accountID <= 0 || id == uuid.Nil {
		return ErrInvalidRequest
	}
	deleted, err := s.store.RevokeOAuthClient(ctx, accountID, id)
	if err != nil {
		return err
	}
	if !deleted {
		return ErrInvalidClient
	}
	return nil
}

func (s *Service) ReviewAuthorization(ctx context.Context, req AuthorizationRequest) (*auth_model.OAuthClient, error) {
	if err := validateAuthorization(req); err != nil {
		return nil, err
	}
	row, err := s.store.GetOAuthClient(ctx, req.ClientID)
	if errors.Is(err, db.ErrOAuthClientMissing) {
		return nil, ErrInvalidClient
	}
	if err != nil {
		return nil, err
	}
	// Validate the public PKCE client before crossing the model boundary.
	// A confidential client must never be treated as a public client.
	if row.RedirectUri != req.RedirectURI || len(row.SecretHash) != 0 {
		return nil, ErrInvalidClient
	}
	return auth_model.OAuthClientFromDB(row), nil
}

// ReviewConsent leaves GET /oauth/authorize read-only. The version is not
// a credential: it prevents approvals based on an obsolete consent preview.
func (s *Service) ReviewConsent(ctx context.Context, accountID int64, req AuthorizationRequest) (*auth_model.OAuthClient, int64, error) {
	if accountID <= 0 {
		return nil, 0, ErrAccessDenied
	}
	client, err := s.ReviewAuthorization(ctx, req)
	if err != nil {
		return nil, 0, err
	}
	version, err := s.store.GetOAuthConsentVersion(ctx, accountID, req.ClientID)
	if err != nil {
		return nil, 0, err
	}
	return client, version, nil
}

func (s *Service) Authorize(ctx context.Context, accountID int64, sessionID uuid.UUID, req AuthorizationRequest) (string, error) {
	if accountID <= 0 || sessionID == uuid.Nil {
		return "", ErrAccessDenied
	}
	if _, err := s.ReviewAuthorization(ctx, req); err != nil {
		return "", err
	}
	code, err := newBearer()
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256([]byte(code))
	if err := s.store.ApproveOAuthAuthCode(ctx, req.ClientID, accountID,
		req.RedirectURI, req.Scope, req.CodeChallenge, digest[:], time.Now().Add(AuthCodeTTL), req.ConsentVersion, sessionID); err != nil {
		if errors.Is(err, db.ErrOAuthConsentStale) {
			return "", ErrInvalidRequest
		}
		if errors.Is(err, db.ErrOAuthClientMissing) {
			return "", ErrInvalidClient
		}
		if errors.Is(err, db.ErrOAuthSessionInactive) {
			return "", ErrAccessDenied
		}
		return "", err
	}
	redirect, err := url.Parse(req.RedirectURI)
	if err != nil {
		return "", fmt.Errorf("parse registered redirect: %w", err)
	}
	query := redirect.Query()
	query.Set("code", code)
	query.Set("state", req.State)
	redirect.RawQuery = query.Encode()
	return redirect.String(), nil
}

func newBearer() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}
