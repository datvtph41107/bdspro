package auth

import (
	"bytes"
	"testing"
)

func TestRecoveryProof(t *testing.T) {
	raw, digest, err := NewRecoveryProof()
	if err != nil {
		t.Fatal(err)
	}
	got, err := RecoveryDigest(raw)
	if err != nil || !bytes.Equal(got, digest) || bytes.Equal([]byte(raw), digest) {
		t.Fatal("invalid recovery digest")
	}
	other, _, err := NewRecoveryProof()
	if err != nil || other == raw {
		t.Fatal("proof generation not unique")
	}
	for _, invalid := range []string{"", raw + "=", "not-a-token"} {
		if _, err := RecoveryDigest(invalid); err == nil {
			t.Fatal("malformed proof accepted")
		}
	}
}
