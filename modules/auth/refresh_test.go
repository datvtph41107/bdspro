package auth

import (
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
