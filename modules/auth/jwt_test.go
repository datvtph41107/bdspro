package auth

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func TestAccessTokenContract(t *testing.T) {
	const key = "disposable-jwt-contract-key-0123456789"
	maker, err := NewJWTMaker(key)
	if err != nil {
		t.Fatal(err)
	}
	sessionID := uuid.New()
	raw, payload, err := maker.CreateSessionAccessToken(7, time.Minute, sessionID, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := maker.VerifyAccessToken(raw)
	if err != nil || decoded.SessionID != sessionID || decoded.AccountID != 7 || decoded.TokenType != "access" {
		t.Fatal("valid session access token rejected", err)
	}
	wire, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(wire, &fields); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"id", "session_id", "account_id", "token_type", "issued_at", "expired_at"} {
		if _, ok := fields[name]; !ok {
			t.Fatal("missing claim", name)
		}
	}
	if len(fields) != 6 {
		t.Fatal("JWT wire contract changed")
	}

	for _, tc := range []struct {
		name   string
		method jwt.SigningMethod
		key    []byte
		edit   func(*Payload)
		want   error
	}{
		{"wrong type", jwt.SigningMethodHS256, []byte(key), func(p *Payload) { p.TokenType = "refresh" }, ErrInvalidToken},
		{"missing session", jwt.SigningMethodHS256, []byte(key), func(p *Payload) { p.SessionID = uuid.Nil }, ErrInvalidToken},
		{"missing token ID", jwt.SigningMethodHS256, []byte(key), func(p *Payload) { p.ID = uuid.Nil }, ErrInvalidToken},
		{"invalid account", jwt.SigningMethodHS256, []byte(key), func(p *Payload) { p.AccountID = 0 }, ErrInvalidToken},
		{"wrong key", jwt.SigningMethodHS256, []byte("another-disposable-key-01234567890123"), func(*Payload) {}, ErrInvalidToken},
		{"HS384", jwt.SigningMethodHS384, []byte(key), func(*Payload) {}, ErrInvalidToken},
		{"expired", jwt.SigningMethodHS256, []byte(key), func(p *Payload) { p.IssuedAt = time.Now().Add(-time.Hour); p.ExpiredAt = time.Now().Add(-time.Minute) }, ErrExpiredToken},
		{"future issuance", jwt.SigningMethodHS256, []byte(key), func(p *Payload) { p.IssuedAt = time.Now().Add(time.Hour); p.ExpiredAt = time.Now().Add(2 * time.Hour) }, ErrInvalidToken},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := *payload
			tc.edit(&p)
			signed, err := jwt.NewWithClaims(tc.method, &p).SignedString(tc.key)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := maker.VerifyAccessToken(signed); !errors.Is(err, tc.want) {
				t.Fatalf("got %v, want %v", err, tc.want)
			}
		})
	}
	if _, err := maker.VerifyAccessToken("invalid.jwt"); !errors.Is(err, ErrInvalidToken) {
		t.Fatal("malformed token accepted")
	}
	if _, _, err := maker.CreateSessionAccessToken(7, time.Minute, uuid.Nil, time.Now().Add(time.Hour)); !errors.Is(err, ErrInvalidToken) {
		t.Fatal("unbound access token issued")
	}
	if _, err := NewJWTMaker("short"); err == nil {
		t.Fatal("short key accepted")
	}
}
