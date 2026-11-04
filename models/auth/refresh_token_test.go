package auth

import (
	"strings"
	"testing"
)

func TestRefreshTokenFingerprintMatchesNewAndLegacyRows(t *testing.T) {
	const raw = "eyJ.mock-valid-jwt-bearer-token"
	digest := RefreshTokenDigest(raw)
	if !strings.HasPrefix(digest, "sha256:") || len(digest) != 71 || strings.Contains(digest, raw) {
		t.Fatalf("unexpected refresh digest format: %q", digest)
	}
	if RefreshTokenDigest(raw) != digest {
		t.Fatal("refresh digest is not stable")
	}
	for _, stored := range []string{raw, digest} {
		if !MatchesRefreshToken(stored, raw) {
			t.Fatalf("valid refresh bearer not matched: %q", stored)
		}
		if MatchesRefreshToken(stored, "wrong-token") ||
			MatchesRefreshToken(stored, "") {
			t.Fatal("invalid refresh bearer matched")
		}
	}
	if MatchesRefreshToken("", raw) || MatchesRefreshToken("sha256:invalid", raw) {
		t.Fatal("empty or malformed persisted digest accepted")
	}
}
