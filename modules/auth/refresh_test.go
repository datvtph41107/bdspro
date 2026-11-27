package auth

import (
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestRefreshCredential(t *testing.T) {
	key := strings.Repeat("k", 32)
	id := uuid.New()
	raw, err := NewRefreshCredential(key, id, true)
	if err != nil {
		t.Fatal(err)
	}
	proof, err := ReadRefreshCredential(key, raw)
	if err != nil || proof.SessionID != id || !proof.Remember {
		t.Fatal("valid refresh proof rejected")
	}
	next, err := NewRefreshCredential(key, id, true)
	if err != nil || raw == next {
		t.Fatal("refresh did not rotate")
	}
	if !MatchesRefreshDigest(raw, RefreshDigest(raw)) || MatchesRefreshDigest(next, RefreshDigest(raw)) || MatchesRefreshDigest(raw, raw) {
		t.Fatal("refresh digest matching failed")
	}
	for _, bad := range []string{raw + "x", strings.Replace(raw, id.String(), uuid.NewString(), 1), strings.Replace(raw, ".1.", ".0.", 1), "legacy-token"} {
		if _, err := ReadRefreshCredential(key, bad); err == nil {
			t.Fatal("invalid proof accepted")
		}
	}
	if _, err := ReadRefreshCredential(strings.Repeat("x", 32), raw); err == nil {
		t.Fatal("wrong key accepted")
	}
}

func TestRefreshWireCompatibility(t *testing.T) {
	const key = "disposable-refresh-contract-key-12345"
	const raw = "r1.12345678-1234-1234-1234-123456789abc.AAECAwQFBgcICQoLDA0ODxAREhMUFRYXGBkaGxwdHh8.1._e7LRnalvti9rOZK81dG5zBHfeuxRQ6-hZk0AYNtecE"
	const digest = "sha256:280532e9f442fb60f0e7c1a8ae6ad641db73275398920cbbc599c76ee0c91645"
	proof, err := ReadRefreshCredential(key, raw)
	if err != nil || proof.SessionID.String() != "12345678-1234-1234-1234-123456789abc" || !proof.Remember {
		t.Fatal("existing refresh credential rejected", err)
	}
	if RefreshDigest(raw) != digest || !MatchesRefreshDigest(raw, digest) {
		t.Fatal("persisted refresh digest format changed")
	}
	// CR/LF in a tag used to decode to the same bytes while changing the
	// persisted raw-string digest. Reject aliases before session lookup.
	for _, bad := range []string{raw + "\n", raw + "\r\n", raw[:len(raw)-1] + "\n" + raw[len(raw)-1:], raw + "="} {
		if _, err := ReadRefreshCredential(key, bad); !errors.Is(err, ErrRefreshInvalid) {
			t.Fatal("noncanonical refresh tag accepted")
		}
	}
}
