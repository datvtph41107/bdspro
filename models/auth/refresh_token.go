package auth

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"strings"
)

const refreshTokenDigestPrefix = "sha256:"

// RefreshTokenDigest stores a one-way fingerprint of a high-entropy signed JWT.
// Clients receive the original bearer token; database sessions persist only this
// fingerprint for newly issued credentials.
func RefreshTokenDigest(raw string) string {
	digest := sha256.Sum256([]byte(raw))
	return refreshTokenDigestPrefix + hex.EncodeToString(digest[:])
}

// MatchesRefreshToken supports both fingerprinted sessions and legacy raw JWT
// rows during a bounded migration window. JWT validity and account/session
// authority are checked separately by the caller before this comparison.
func MatchesRefreshToken(stored, raw string) bool {
	if stored == "" || raw == "" {
		return false
	}
	expected := raw
	if strings.HasPrefix(stored, refreshTokenDigestPrefix) {
		expected = RefreshTokenDigest(raw)
	}
	return subtle.ConstantTimeCompare([]byte(stored), []byte(expected)) == 1
}
