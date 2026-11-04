package secret

import (
	"crypto/sha256"
	"testing"

	"github.com/google/uuid"
)

func TestDeriveToken(t *testing.T) {
	id := uuid.New()
	key := "01234567890123456789012345678901"
	first, err := DeriveToken(key, "team-invite:v1", id[:])
	if err != nil || len(first) != 64 {
		t.Fatalf("derive token: size=%d err=%v", len(first), err)
	}
	second, err := DeriveToken(key, "team-invite:v1", id[:])
	if err != nil || second != first {
		t.Fatal("durable task could not reproduce the invitation proof")
	}
	other, _ := DeriveToken(key, "password-reset:v1", id[:])
	if other == first {
		t.Fatal("two token purposes produced identical proofs")
	}
	hash := sha256.Sum256([]byte(first))
	if string(hash[:]) == first {
		t.Fatal("plaintext should not be persisted as a hash")
	}
	if _, err := DeriveToken("weak", "team-invite:v1", id[:]); err == nil {
		t.Fatal("short signing key accepted")
	}
}
