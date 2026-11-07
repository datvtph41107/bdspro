package auth

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func TestJWTAccessRefreshTypeAndSignature(t *testing.T) {
	maker, err := NewJWTMaker(strings.Repeat("k", 32))
	if err != nil {
		t.Fatal(err)
	}
	access, issued, err := maker.CreateToken(123, time.Minute, TokenTypeAccess)
	if err != nil {
		t.Fatal(err)
	}
	got, err := maker.VerifyToken(access, TokenTypeAccess)
	if err != nil || got.ID != issued.ID || got.AccountID != 123 {
		t.Fatalf("access token no longer verifies: %+v, %v", got, err)
	}
	if _, err := maker.VerifyToken(access, TokenTypeRefresh); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("access token accepted as refresh: %v", err)
	}
	if _, err := maker.VerifyToken(access+"x", TokenTypeAccess); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("modified signature accepted: %v", err)
	}
	if _, err := NewJWTMaker("short"); err == nil {
		t.Fatal("weak signing key accepted")
	}
}

func TestJWTLegacyWireClaimsRemainReadable(t *testing.T) {
	secret := strings.Repeat("k", 32)
	maker, err := NewJWTMaker(secret)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Truncate(time.Second)
	claims := &Payload{
		ID:        uuid.MustParse("00000000-0000-4000-8000-000000000123"),
		AccountID: 19,
		TokenType: TokenTypeRefresh,
		IssuedAt:  now.Add(-time.Minute),
		ExpiredAt: now.Add(time.Hour),
	}
	legacyWire, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	if err != nil {
		t.Fatal(err)
	}
	got, err := maker.VerifyToken(legacyWire, TokenTypeRefresh)
	if err != nil || got.AccountID != claims.AccountID || got.ID != claims.ID ||
		!got.ExpiredAt.Equal(claims.ExpiredAt) {
		t.Fatalf("legacy JWT compatibility broken: %+v, %v", got, err)
	}
	claims.ExpiredAt = now.Add(-time.Second)
	expired, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := maker.VerifyToken(expired, TokenTypeRefresh); !errors.Is(err, ErrExpiredToken) {
		t.Fatalf("expired JWT accepted: %v", err)
	}
}

// OAuth authorization requires a session-bound access token. Generic access
// tokens and legacy signed claims remain valid for existing stateless APIs.
func TestJWTSessionBoundAccessToken(t *testing.T) {
	maker, err := NewJWTMaker(strings.Repeat("k", 32))
	if err != nil { t.Fatal(err) }
	sid := uuid.New()
	raw, issued, err := maker.CreateSessionAccessToken(42, time.Minute, sid)
	if err != nil { t.Fatal(err) }
	verified, err := maker.VerifyToken(raw, TokenTypeAccess)
	if err != nil || verified.SessionID != sid || issued.SessionID != sid || issued.ID == sid {
		t.Fatalf("bound token lost session identity: %+v, %+v, %v", issued, verified, err)
	}
	if _, _, err := maker.CreateSessionAccessToken(42, time.Minute, uuid.Nil); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("accepted access token without session: %v", err)
	}
	legacy, _, err := maker.CreateToken(42, time.Minute, TokenTypeAccess)
	if err != nil { t.Fatal(err) }
	got, err := maker.VerifyToken(legacy, TokenTypeAccess)
	if err != nil || got.SessionID != uuid.Nil {
		t.Fatalf("legacy stateless JWT compatibility broken: %+v, %v", got, err)
	}
}
