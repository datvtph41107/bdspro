package auth

import (
	"errors"
	"strings"
	"testing"
)

func TestOIDCIdentityKey(t *testing.T) {
	tests := []struct {
		name     string
		identity OIDCIdentity
		wantErr  error
	}{
		{name: "valid", identity: OIDCIdentity{Issuer: "https://accounts.example.test", Subject: "subject-1"}},
		{name: "issuer with path", identity: OIDCIdentity{Issuer: "https://example.test/tenant", Subject: "subject-1"}},
		{name: "insecure issuer", identity: OIDCIdentity{Issuer: "http://example.test", Subject: "subject-1"}, wantErr: ErrOIDCIssuerInvalid},
		{name: "query issuer", identity: OIDCIdentity{Issuer: "https://example.test?x=1", Subject: "subject-1"}, wantErr: ErrOIDCIssuerInvalid},
		{name: "empty subject", identity: OIDCIdentity{Issuer: "https://example.test", Subject: " "}, wantErr: ErrOIDCSubjectInvalid},
		{name: "non-ascii subject", identity: OIDCIdentity{Issuer: "https://example.test", Subject: "émail"}, wantErr: ErrOIDCSubjectInvalid},
		{name: "long subject", identity: OIDCIdentity{Issuer: "https://example.test", Subject: strings.Repeat("x", 256)}, wantErr: ErrOIDCSubjectInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.identity.Key()
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Key error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr == nil && (got.Issuer != tt.identity.Issuer || got.Subject != tt.identity.Subject) {
				t.Fatalf("Key = %#v", got)
			}
		})
	}
}

func TestOIDCIdentityKeyDoesNotUseEmail(t *testing.T) {
	first := OIDCIdentity{Issuer: "https://first.example.test", Subject: "same", Email: "a@example.test", EmailVerified: true}
	second := OIDCIdentity{Issuer: "https://second.example.test", Subject: "same", Email: "a@example.test", EmailVerified: true}
	a, err := first.Key()
	if err != nil {
		t.Fatal(err)
	}
	b, err := second.Key()
	if err != nil {
		t.Fatal(err)
	}
	if a == b {
		t.Fatal("identities from different issuers must not collide")
	}
}
