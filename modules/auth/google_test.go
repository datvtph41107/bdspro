package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"net/url"
	"testing"
	"time"

	"github.com/datvtph41107/bdspro/modules/setting"
	"github.com/golang-jwt/jwt/v5"
)

func TestGoogleAuthorizationURL(t *testing.T) {
	cfg := setting.GoogleSettings{ClientID: "test-client", CallbackURL: "http://localhost:5173/api/v1/auth/google/callback"}
	u, err := url.Parse(GoogleAuthorizationURL(cfg, "state", "nonce", "verifier", false))
	if err != nil || u.Host != "accounts.google.com" || u.Query().Get("code_challenge_method") != "S256" ||
		u.Query().Get("code_challenge") != "iMnq5o6zALKXGivsnlom_0F5_WYda32GHkxlV7mq7hQ" || u.Query().Get("redirect_uri") != cfg.CallbackURL ||
		u.Query().Get("state") != "state" || u.Query().Get("nonce") != "nonce" {
		t.Fatal("invalid authorization request")
	}
}

func TestGoogleClaims(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	valid := googleClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer: GoogleIssuer, Subject: "stable-subject", Audience: jwt.ClaimStrings{"client"},
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Minute)), IssuedAt: jwt.NewNumericDate(now),
		},
		Nonce: "nonce", Email: "owner@gmail.com", EmailVerified: true, AuthTime: now.Unix(),
	}
	for _, tc := range []struct {
		name   string
		change func(*googleClaims)
		valid  bool
		auth   bool
	}{
		{"valid", func(*googleClaims) {}, true, true},
		{"third party email", func(c *googleClaims) { c.Email = "user@example.invalid" }, true, false},
		{"unverified", func(c *googleClaims) { c.EmailVerified = false }, true, false},
		{"issuer", func(c *googleClaims) { c.Issuer = "https://example.invalid" }, false, false},
		{"audience", func(c *googleClaims) { c.Audience = jwt.ClaimStrings{"wrong"} }, false, false},
		{"azp", func(c *googleClaims) { c.Authorized = "wrong" }, false, false},
		{"nonce", func(c *googleClaims) { c.Nonce = "wrong" }, false, false},
		{"expired", func(c *googleClaims) { c.ExpiresAt = jwt.NewNumericDate(now.Add(-time.Minute)) }, false, false},
		{"subject", func(c *googleClaims) { c.Subject = "" }, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := valid
			tc.change(&c)
			raw, err := jwt.NewWithClaims(jwt.SigningMethodRS256, &c).SignedString(key)
			if err != nil {
				t.Fatal(err)
			}
			identity, err := verifyGoogleIDToken(raw, "client", "nonce", func(*jwt.Token) (any, error) { return &key.PublicKey, nil })
			if (err == nil) != tc.valid || tc.valid && (identity.AuthoritativeEmail != tc.auth || identity.AuthTime != c.AuthTime) {
				t.Fatal("incorrect claim verification")
			}
		})
	}
}
