package auth

import (
	"context"
	"crypto/rsa"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"math/big"
	"net/http"
	"time"

	auth_model "github.com/datvtph41107/bdspro/models/auth"
	"github.com/golang-jwt/jwt/v5"
)

// OIDCTokenVerifier verifies ID tokens against one explicitly trusted issuer.
// An OAuth2 access token is NOT an identity assertion.
type OIDCTokenVerifier struct {
	Issuer   string
	Audience string
	JWKSURL  string
	Client   *http.Client
}

type oidcIDClaims struct {
	jwt.RegisteredClaims
	Nonce         string `json:"nonce"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
}

// Verify validates signature, algorithm, expected issuer, audience, exp, iat
// and browser-bound nonce. Only RS256 is supported by this implementation.
// The caller must supply metadata from a trusted, configured OIDC issuer.
func (v OIDCTokenVerifier) Verify(ctx context.Context, raw, expectedNonce string) (auth_model.OIDCIdentity, error) {
	var empty auth_model.OIDCIdentity
	if raw == "" || v.Issuer == "" || v.Audience == "" || v.JWKSURL == "" ||
		v.Client == nil || expectedNonce == "" {
		return empty, errors.New("invalid OpenID Connect verification parameters")
	}
	if _, err := (auth_model.OIDCIdentity{Issuer: v.Issuer, Subject: "probe"}).Key(); err != nil {
		return empty, err
	}
	claims := &oidcIDClaims{}
	token, err := jwt.NewParser(
		jwt.WithValidMethods([]string{"RS256"}),
		jwt.WithAudience(v.Audience),
		jwt.WithIssuer(v.Issuer),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
		jwt.WithLeeway(30*time.Second),
	).ParseWithClaims(raw, claims, func(token *jwt.Token) (any, error) {
		kid, ok := token.Header["kid"].(string)
		if !ok || kid == "" {
			return nil, errors.New("missing OIDC verification key ID")
		}
		return fetchOIDCRSAKey(ctx, v.Client, v.JWKSURL, kid)
	})
	if err != nil || token == nil || !token.Valid {
		return empty, errors.New("invalid OIDC ID token")
	}
	if claims.Subject == "" || subtle.ConstantTimeCompare([]byte(claims.Nonce), []byte(expectedNonce)) != 1 {
		return empty, errors.New("invalid OIDC identity claims")
	}
	result := auth_model.OIDCIdentity{
		Issuer: claims.Issuer, Subject: claims.Subject,
		Email: claims.Email, EmailVerified: claims.EmailVerified,
	}
	if _, err := result.Key(); err != nil {
		return empty, err
	}
	return result, nil
}

type oidcJWKSet struct {
	Keys []struct {
		Kty string `json:"kty"`
		Kid string `json:"kid"`
		Alg string `json:"alg"`
		Use string `json:"use"`
		N   string `json:"n"`
		E   string `json:"e"`
	} `json:"keys"`
}

func fetchOIDCRSAKey(ctx context.Context, client *http.Client, uri, kid string) (*rsa.PublicKey, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri, nil)
	if err != nil {
		return nil, err
	}
	bounded := *client
	if bounded.Timeout == 0 || bounded.Timeout > 10*time.Second {
		bounded.Timeout = 10 * time.Second
	}
	res, err := bounded.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, errors.New("OIDC public keys unavailable")
	}
	var keys oidcJWKSet
	if err := json.NewDecoder(io.LimitReader(res.Body, 64<<10)).Decode(&keys); err != nil {
		return nil, err
	}
	for _, key := range keys.Keys {
		if key.Kid != kid || key.Kty != "RSA" || key.Alg != "RS256" || (key.Use != "" && key.Use != "sig") {
			continue
		}
		n, err := base64.RawURLEncoding.DecodeString(key.N)
		if err != nil {
			return nil, err
		}
		e, err := base64.RawURLEncoding.DecodeString(key.E)
		if err != nil {
			return nil, err
		}
		exponent := new(big.Int).SetBytes(e)
		if len(n) < 256 || exponent.Sign() <= 0 || exponent.BitLen() > 31 ||
			exponent.Int64() < 3 || exponent.Int64()%2 == 0 {
			return nil, errors.New("invalid OIDC RSA key")
		}
		return &rsa.PublicKey{N: new(big.Int).SetBytes(n), E: int(exponent.Int64())}, nil
	}
	return nil, errors.New("OIDC verification key not found")
}
