package auth

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/google/uuid"
)

func TestOAuthClientProjectionNeverExportsSecretHash(t *testing.T) {
	const secret = "PRIVATE_OAUTH_CLIENT_SECRET_HASH"
	now := time.Date(2026, 11, 10, 8, 0, 0, 0, time.UTC)
	row := db.OauthClient{
		ID: uuid.New(), OwnerAccountID: 41, Name: "Partner",
		RedirectUri: "https://client.example.test/callback",
		SecretHash:  []byte(secret), CreatedAt: now,
	}
	model := OAuthClientFromDB(row)
	if model.ID != row.ID || model.OwnerAccountID != 41 ||
		model.Name != "Partner" || model.RedirectURI != row.RedirectUri ||
		model.CreatedAt != now {
		t.Fatalf("OAuth client metadata changed: %+v", model)
	}
	data, err := json.Marshal(model)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), secret) || strings.Contains(string(data), "SecretHash") {
		t.Fatalf("OAuth client domain leaked secret: %s", data)
	}
	row.Name = "mutated"
	row.SecretHash[0] = 'X'
	if model.Name != "Partner" {
		t.Fatal("OAuth model aliases mutable SQLC row")
	}
}

func TestOAuth2CredentialProjectionIsModelOwned(t *testing.T) {
	now := time.Date(2026, 11, 10, 8, 0, 0, 0, time.UTC)
	row := db.Oauth2Credential{Issuer: "https://issuer.example.test", Subject: "subject", AccountID: 7, LinkedAt: now}
	model := oauth2CredentialFromDB(row)
	if model.Issuer != row.Issuer || model.Subject != row.Subject ||
		model.AccountID != row.AccountID || model.LinkedAt != now {
		t.Fatalf("OAuth credential metadata changed: %+v", model)
	}
}
