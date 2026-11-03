package secret

import (
	"encoding/hex"
	"testing"
)

func TestRandomHexIsCryptoSized(t *testing.T) {
	a, err := RandomHex(32)
	if err != nil {
		t.Fatal(err)
	}
	b, err := RandomHex(32)
	if err != nil {
		t.Fatal(err)
	}
	if a == b {
		t.Fatal("repeated secret")
	}
	decoded, err := hex.DecodeString(a)
	if err != nil || len(decoded) != 32 {
		t.Fatalf("secret bytes: %d %v", len(decoded), err)
	}
	if _, err := RandomHex(0); err == nil {
		t.Fatal("zero length accepted")
	}
}
