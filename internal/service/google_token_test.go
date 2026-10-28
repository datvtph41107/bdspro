package service

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

func TestVerifyGoogleIDTokenChecksIdentityAndNonce(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	encoded := func(n []byte) string { return base64.RawURLEncoding.EncodeToString(n) }
	jwks := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"keys": []map[string]string{{
				"kty": "RSA", "kid": "test-kid", "alg": "RS256", "use": "sig",
				"n": encoded(key.PublicKey.N.Bytes()),
				"e": encoded(big.NewInt(int64(key.PublicKey.E)).Bytes()),
			}},
		})
	}))
	defer jwks.Close()

	claims := &googleIDClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "https://accounts.google.com",
			Subject:   "google-subject",
			Audience:  jwt.ClaimStrings{"test-client"},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(10 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-time.Minute)),
		},
		Nonce:         "random-browser-nonce",
		Email:         "verified@example.test",
		EmailVerified: true,
	}
	sign := func() string {
		t.Helper()
		token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
		token.Header["kid"] = "test-kid"
		signed, err := token.SignedString(key)
		if err != nil {
			t.Fatal(err)
		}
		return signed
	}
	raw := sign()
	identity, err := verifyGoogleIDToken(
		context.Background(), jwks.Client(), jwks.URL,
		raw, "test-client", "random-browser-nonce",
	)
	if err != nil {
		t.Fatal(err)
	}
	if identity.Subject != "google-subject" || identity.Email != "verified@example.test" ||
		!identity.EmailVerified {
		t.Fatalf("verified claims changed: %+v", identity)
	}
	for _, tc := range []struct{ name, client, nonce string }{
		{"wrong audience", "other-client", "random-browser-nonce"},
		{"wrong nonce", "test-client", "different-nonce"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := verifyGoogleIDToken(context.Background(), jwks.Client(), jwks.URL, raw, tc.client, tc.nonce); err == nil {
				t.Fatal("incorrect Google identity assertion was accepted")
			}
		})
	}
	claims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(-2 * time.Minute))
	if _, err := verifyGoogleIDToken(context.Background(), jwks.Client(), jwks.URL, sign(), "test-client", "random-browser-nonce"); err == nil {
		t.Fatal("expired Google token was accepted")
	}
}
