package oauth

import (
	"strings"
	"testing"
	"time"
)

func testConsentProofFields() ConsentProofFields {
	return ConsentProofFields{
		AccountID: 42, ClientID: "4df5bcd8-2cd2-496d-9ec1-cd4f99d979ce",
		RedirectURI: "https://client.example.test/callback", Scope: "profile:read",
		ResponseType: "code", CodeChallenge: strings.Repeat("x", 43),
		CodeChallengeMethod: "S256", State: "0123456789abcdef", ConsentVersion: 3,
	}
}

func TestConsentProofBoundToPreviewAndExpiry(t *testing.T) {
	signer, err := NewConsentProofs(strings.Repeat("s", 32))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 7, 0, 0, 0, 0, time.UTC)
	base := testConsentProofFields()
	token, err := signer.Issue(base, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := signer.Verify(token, base, now.Add(4*time.Minute)); err != nil {
		t.Fatal(err)
	}
	cases := map[string]func(*ConsentProofFields){
		"account":   func(f *ConsentProofFields) { f.AccountID++ },
		"client":    func(f *ConsentProofFields) { f.ClientID = "00000000-0000-0000-0000-000000000001" },
		"redirect":  func(f *ConsentProofFields) { f.RedirectURI = "https://other.example.test/callback" },
		"scope":     func(f *ConsentProofFields) { f.Scope = "admin" },
		"response":  func(f *ConsentProofFields) { f.ResponseType = "token" },
		"challenge": func(f *ConsentProofFields) { f.CodeChallenge = "different" },
		"method":    func(f *ConsentProofFields) { f.CodeChallengeMethod = "plain" },
		"state":     func(f *ConsentProofFields) { f.State = "different" },
		"version":   func(f *ConsentProofFields) { f.ConsentVersion++ },
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			changed := base
			change(&changed)
			if err := signer.Verify(token, changed, now); err == nil {
				t.Fatal("changed consent accepted")
			}
		})
	}
	if err := signer.Verify(token, base, now.Add(5*time.Minute)); err == nil {
		t.Fatal("expired consent accepted")
	}
	if err := signer.Verify(token, base, now.Add(-1*time.Minute)); err == nil {
		t.Fatal("future consent accepted")
	}
	if err := signer.Verify(token+"x", base, now); err == nil {
		t.Fatal("tampered proof accepted")
	}
	other, err := NewConsentProofs(strings.Repeat("b", 32))
	if err != nil {
		t.Fatal(err)
	}
	if err := other.Verify(token, base, now); err == nil {
		t.Fatal("wrong signing key accepted")
	}
}

func TestConsentProofRejectsMalformedAndWeakSecret(t *testing.T) {
	if _, err := NewConsentProofs("weak"); err == nil {
		t.Fatal("short secret accepted")
	}
	signer, err := NewConsentProofs(strings.Repeat("s", 32))
	if err != nil {
		t.Fatal(err)
	}
	fields := testConsentProofFields()
	for _, token := range []string{"", "v1.bad", "v2.bad.bad", strings.Repeat("A", 8193)} {
		if err := signer.Verify(token, fields, time.Now()); err == nil {
			t.Fatalf("accepted %q", token[:min(len(token), 25)])
		}
	}
}
