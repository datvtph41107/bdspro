package service

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"math/big"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const googleJWKSURL = "https://www.googleapis.com/oauth2/v3/certs"

// VerifiedGoogleIdentity contains claims from a validated Google ID token.
type VerifiedGoogleIdentity struct {
	Subject       string
	Email         string
	EmailVerified bool
}

type googleIDClaims struct {
	jwt.RegisteredClaims
	Nonce         string `json:"nonce"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
}

// VerifyGoogleIDToken checks Google's RSA signature, audience, expiry, issuer
// and the nonce bound to the user's one-time browser authorization flow.
func VerifyGoogleIDToken(ctx context.Context, raw, clientID, nonce string) (VerifiedGoogleIdentity, error) {
	return verifyGoogleIDToken(ctx, http.DefaultClient, googleJWKSURL, raw, clientID, nonce)
}

func verifyGoogleIDToken(ctx context.Context, client *http.Client, jwksURL, raw, clientID, nonce string) (VerifiedGoogleIdentity, error) {
	var empty VerifiedGoogleIdentity
	if raw == "" || clientID == "" || nonce == "" {
		return empty, errors.New("invalid Google identity assertion")
	}
	claims := &googleIDClaims{}
	tok, err := jwt.NewParser(
		jwt.WithValidMethods([]string{"RS256"}),
		jwt.WithAudience(clientID),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
		jwt.WithLeeway(30*time.Second),
	).ParseWithClaims(raw, claims, func(token *jwt.Token) (any, error) {
		kid, ok := token.Header["kid"].(string)
		if !ok || kid == "" {
			return nil, errors.New("missing Google key ID")
		}
		return fetchGoogleRSAKey(ctx, client, jwksURL, kid)
	})
	if err != nil || !tok.Valid {
		return empty, errors.New("invalid Google ID token")
	}
	if (claims.Issuer != "https://accounts.google.com" && claims.Issuer != "accounts.google.com") ||
		claims.Subject == "" || claims.Nonce != nonce {
		return empty, errors.New("invalid Google identity claims")
	}
	return VerifiedGoogleIdentity{
		Subject:       claims.Subject,
		Email:         claims.Email,
		EmailVerified: claims.EmailVerified,
	}, nil
}

type googleJWKSet struct {
	Keys []struct {
		Kty string `json:"kty"`
		Kid string `json:"kid"`
		Alg string `json:"alg"`
		Use string `json:"use"`
		N   string `json:"n"`
		E   string `json:"e"`
	} `json:"keys"`
}

func fetchGoogleRSAKey(ctx context.Context, client *http.Client, uri, kid string) (*rsa.PublicKey, error) {
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
		return nil, errors.New("Google public keys unavailable")
	}
	var keys googleJWKSet
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
			return nil, errors.New("invalid Google RSA key")
		}
		return &rsa.PublicKey{N: new(big.Int).SetBytes(n), E: int(exponent.Int64())}, nil
	}
	return nil, errors.New("Google verification key not found")
}
