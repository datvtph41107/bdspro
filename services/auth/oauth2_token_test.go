package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestOIDCTokenVerifierRejectsWrongIssuerAndNonce(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	encode := base64.RawURLEncoding.EncodeToString
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []map[string]string{{
			"kty": "RSA", "kid": "kid-1", "alg": "RS256", "use": "sig",
			"n": encode(key.PublicKey.N.Bytes()),
			"e": encode(big.NewInt(int64(key.PublicKey.E)).Bytes()),
		}}})
	}))
	defer server.Close()
	issuer := "https://trusted.example.test/tenant"
	claims := &oidcIDClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer: issuer, Subject: "tenant-user-1",
			Audience:  jwt.ClaimStrings{"bdspro-client"},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(5 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-time.Minute)),
		},
		Nonce:         "browser-nonce",
		Email:         "verified@example.test",
		EmailVerified: true,
	}
	sign := func() string {
		t.Helper()
		tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
		tok.Header["kid"] = "kid-1"
		value, err := tok.SignedString(key)
		if err != nil {
			t.Fatal(err)
		}
		return value
	}
	verifier := OIDCTokenVerifier{
		Issuer: issuer, Audience: "bdspro-client",
		JWKSURL: server.URL, Client: server.Client(),
	}
	id, err := verifier.Verify(context.Background(), sign(), "browser-nonce")
	if err != nil || id.Issuer != issuer || id.Subject != "tenant-user-1" {
		t.Fatalf("valid ID token rejected: %#v %v", id, err)
	}
	for _, tc := range []struct {
		name  string
		check func() error
	}{
		{"wrong nonce", func() error { _, err := verifier.Verify(context.Background(), sign(), "wrong"); return err }},
		{"multiple audiences without authorized party", func() error {
			original := claims.Audience
			claims.Audience = jwt.ClaimStrings{"bdspro-client", "other-client"}
			defer func() { claims.Audience = original }()
			_, err := verifier.Verify(context.Background(), sign(), "browser-nonce")
			return err
		}},
		{"wrong issuer", func() error {
			old := claims.Issuer
			claims.Issuer = "https://attacker.example.test"
			defer func() { claims.Issuer = old }()
			_, err := verifier.Verify(context.Background(), sign(), "browser-nonce")
			return err
		}},
		{"wrong audience", func() error {
			copy := verifier
			copy.Audience = "other-client"
			_, err := copy.Verify(context.Background(), sign(), "browser-nonce")
			return err
		}},
		{"missing issued-at", func() error {
			old := claims.IssuedAt
			claims.IssuedAt = nil
			defer func() { claims.IssuedAt = old }()
			_, err := verifier.Verify(context.Background(), sign(), "browser-nonce")
			return err
		}},
		{"expired", func() error {
			old := claims.ExpiresAt
			claims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(-2 * time.Minute))
			defer func() { claims.ExpiresAt = old }()
			_, err := verifier.Verify(context.Background(), sign(), "browser-nonce")
			return err
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.check(); err == nil {
				t.Fatal("invalid OIDC ID token was accepted")
			}
		})
	}
}
