package oauth

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"testing"
	"time"

	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/google/uuid"
)

type clientDomainStore struct {
	db.OAuthServerStore
	row db.OauthClient
	consentCalls int
}

func (s *clientDomainStore) CreateOAuthClient(_ context.Context, row db.OauthClient) (db.OauthClient, error) {
	row.CreatedAt = s.row.CreatedAt
	row.SecretHash = []byte("PERSISTENCE_ONLY_SECRET")
	return row, nil
}

func (s *clientDomainStore) ListOAuthClients(_ context.Context, _ int64) ([]db.OauthClient, error) {
	return []db.OauthClient{s.row}, nil
}

func (s *clientDomainStore) GetOAuthClient(_ context.Context, id uuid.UUID) (db.OauthClient, error) {
	if id != s.row.ID {
		return db.OauthClient{}, db.ErrOAuthClientMissing
	}
	return s.row, nil
}

func (s *clientDomainStore) GetOAuthConsentVersion(_ context.Context, _ int64, _ uuid.UUID) (int64, error) {
	s.consentCalls++
	return 2, nil
}

func TestOAuthServerOnlyReturnsDomainClient(t *testing.T) {
	now := time.Date(2026, 11, 10, 8, 0, 0, 0, time.UTC)
	id := uuid.New()
	store := &clientDomainStore{row: db.OauthClient{
		ID: id, OwnerAccountID: 7, Name: "Partner",
		RedirectUri: "https://client.example.test/callback", CreatedAt: now,
		SecretHash: []byte("PERSISTENCE_ONLY_SECRET"),
	}}
	svc := NewService(store)
	ctx := context.Background()
	created, err := svc.RegisterClient(ctx, 7, "New client", store.row.RedirectUri)
	if err != nil || created == nil || created.Name != "New client" || created.RedirectURI != store.row.RedirectUri {
		t.Fatalf("OAuth CreateClient projection failed: %+v, %v", created, err)
	}
	list, err := svc.ListClients(ctx, 7)
	if err != nil || len(list) != 1 || list[0].ID != id || list[0].RedirectURI != store.row.RedirectUri {
		t.Fatalf("OAuth ListClients projection failed: %+v, %v", list, err)
	}
	// Public consent must reject a confidential application before any proof
	// can be minted or an authorization code created.
	challenge := sha256.Sum256([]byte("pkce-verifier"))
	req := AuthorizationRequest{
		ClientID: id, RedirectURI: store.row.RedirectUri, ResponseType: "code",
		Scope: ScopeProfile, CodeChallengeMethod: "S256",
		CodeChallenge: base64.RawURLEncoding.EncodeToString(challenge[:]),
		State: "state_0123456789abcdef",
	}
	if client, err := svc.ReviewAuthorization(ctx, req); !errors.Is(err, ErrInvalidClient) || client != nil {
		t.Fatalf("confidential client was admitted: %+v, %v", client, err)
	}
	if client, _, err := svc.ReviewConsent(ctx, 7, req); !errors.Is(err, ErrInvalidClient) ||
		client != nil || store.consentCalls != 0 {
		t.Fatalf("invalid client reached consent lookup: %+v, %v", client, err)
	}
	store.row.SecretHash = nil
	client, err := svc.ReviewAuthorization(ctx, req)
	if err != nil || client == nil || client.ID != id {
		t.Fatalf("public client review failed: %+v, %v", client, err)
	}
	client, version, err := svc.ReviewConsent(ctx, 7, req)
	if err != nil || client == nil || version != 2 || store.consentCalls != 1 {
		t.Fatalf("public client consent review failed: %+v, version=%d, %v", client, version, err)
	}
}
