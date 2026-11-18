package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestOpaqueRefreshTokenIsRandomAndStoredAsDigest(t *testing.T) {
	seen := map[string]struct{}{}
	for i := 0; i < 100; i++ {
		plain, digest, err := newOpaqueRefreshToken()
		if err != nil {
			t.Fatal(err)
		}
		if len(plain) != 43 || len(digest) != 64 {
			t.Fatalf("unexpected credential lengths %d/%d", len(plain), len(digest))
		}
		if plain == digest {
			t.Fatal("plaintext refresh credential stored in DB")
		}
		if _, ok := seen[plain]; ok {
			t.Fatal("duplicate random refresh credential")
		}
		seen[plain] = struct{}{}
		hash := sha256.Sum256([]byte(plain))
		if got := hex.EncodeToString(hash[:]); got != digest {
			t.Fatal("refresh digest mismatch")
		}
	}
}
