package auth

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestAccessTokensIssueAndVerify(t *testing.T) {
	signer, err := NewAccessTokens(strings.Repeat("S", 32))
	if err != nil { t.Fatal(err) }
	token, expiry, err := signer.Issue(17, "opaque-session-example")
	if err != nil { t.Fatal(err) }
	if time.Until(expiry) > 15*time.Minute || time.Until(expiry) < 14*time.Minute {
		t.Fatalf("unexpected expiration %v", expiry)
	}
	id, hash, err := signer.Verify(token)
	if err != nil || id != 17 || len(hash) != 64 { t.Fatalf("verify: %d, %q, %v",id,hash,err) }
	other, err := NewAccessTokens(strings.Repeat("T",32))
	if err != nil { t.Fatal(err) }
	if _, _, err := other.Verify(token); !errors.Is(err,ErrAccessToken) {
		t.Fatalf("wrong key accepted: %v",err)
	}
	if _, _, err := signer.Verify(token+"tampered"); !errors.Is(err,ErrAccessToken) {
		t.Fatalf("tampering accepted: %v",err)
	}
	if _, err := NewAccessTokens("short"); err == nil { t.Fatal("short secret accepted") }
}

func TestAccessTokensRejectWrongClaimsAndAlgorithm(t *testing.T) {
	signer,err:=NewAccessTokens(strings.Repeat("K",32))
	if err!=nil {t.Fatal(err)}
	now:=time.Now().UTC()
	cases:=[]struct{
		name string
		claims accessClaims
	}{
		{"wrong_issuer", accessClaims{TokenType:"access", SessionHash:strings.Repeat("f",64),RegisteredClaims:jwt.RegisteredClaims{Issuer:"attacker",Audience:jwt.ClaimStrings{accessAudience},Subject:"3",IssuedAt:jwt.NewNumericDate(now),NotBefore:jwt.NewNumericDate(now),ExpiresAt:jwt.NewNumericDate(now.Add(time.Minute)),ID:"nonce"}}},
		{"wrong_audience", accessClaims{TokenType:"access", SessionHash:strings.Repeat("f",64),RegisteredClaims:jwt.RegisteredClaims{Issuer:accessIssuer,Audience:jwt.ClaimStrings{"other-api"},Subject:"3",IssuedAt:jwt.NewNumericDate(now),NotBefore:jwt.NewNumericDate(now),ExpiresAt:jwt.NewNumericDate(now.Add(time.Minute)),ID:"nonce"}}},
		{"wrong_token_type", accessClaims{TokenType:"refresh", SessionHash:strings.Repeat("f",64),RegisteredClaims:jwt.RegisteredClaims{Issuer:accessIssuer,Audience:jwt.ClaimStrings{accessAudience},Subject:"3",IssuedAt:jwt.NewNumericDate(now),NotBefore:jwt.NewNumericDate(now),ExpiresAt:jwt.NewNumericDate(now.Add(time.Minute)),ID:"nonce"}}},
		{"expired", accessClaims{TokenType:"access", SessionHash:strings.Repeat("f",64),RegisteredClaims:jwt.RegisteredClaims{Issuer:accessIssuer,Audience:jwt.ClaimStrings{accessAudience},Subject:"3",IssuedAt:jwt.NewNumericDate(now.Add(-time.Hour)),NotBefore:jwt.NewNumericDate(now.Add(-time.Hour)),ExpiresAt:jwt.NewNumericDate(now.Add(-time.Minute)),ID:"nonce"}}},
		{"excessive_lifetime", accessClaims{TokenType:"access", SessionHash:strings.Repeat("f",64),RegisteredClaims:jwt.RegisteredClaims{Issuer:accessIssuer,Audience:jwt.ClaimStrings{accessAudience},Subject:"3",IssuedAt:jwt.NewNumericDate(now),NotBefore:jwt.NewNumericDate(now),ExpiresAt:jwt.NewNumericDate(now.Add(24*time.Hour)),ID:"nonce"}}},
	}
	for _,tc:=range cases {
		t.Run(tc.name,func(t *testing.T){
			raw,err:=jwt.NewWithClaims(jwt.SigningMethodHS256,tc.claims).SignedString(signer.key)
			if err!=nil {t.Fatal(err)}
			if _,_,err:=signer.Verify(raw);!errors.Is(err,ErrAccessToken){t.Fatalf("invalid claims accepted: %v",err)}
		})
	}
	valid,_,err:=signer.Issue(3,"session")
	if err!=nil {t.Fatal(err)}
	_ = valid
	foreign:=jwt.NewWithClaims(jwt.SigningMethodHS384, jwt.MapClaims{"sub":"3","exp":time.Now().Add(time.Minute).Unix()})
	raw,err:=foreign.SignedString(signer.key)
	if err!=nil {t.Fatal(err)}
	if _,_,err:=signer.Verify(raw);!errors.Is(err,ErrAccessToken){t.Fatalf("HS384 accepted: %v",err)}
}
