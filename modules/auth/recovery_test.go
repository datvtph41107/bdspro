package auth

import (
	"bytes"
	"encoding/hex"
	"testing"
)

func TestOpaqueProof(t *testing.T) {
	raw, digest, err := NewOpaqueProof()
	if err != nil {
		t.Fatal(err)
	}
	got, err := OpaqueProofDigest(raw)
	if err != nil || !bytes.Equal(got, digest) || bytes.Equal([]byte(raw), digest) {
		t.Fatal("invalid recovery digest")
	}
	other, _, err := NewOpaqueProof()
	if err != nil || other == raw {
		t.Fatal("proof generation not unique")
	}
	for _, invalid := range []string{"", raw + "=", "not-a-token"} {
		if _, err := OpaqueProofDigest(invalid); err == nil {
			t.Fatal("malformed proof accepted")
		}
	}
}

func TestOpaqueProofWireCompatibility(t *testing.T) {
	const raw = "AAECAwQFBgcICQoLDA0ODxAREhMUFRYXGBkaGxwdHh8"
	const want = "ea866a757e4c38babfa8127cbe9a409d3e1f93a00ff1488ff735fcf917afffd0"
	got, err := OpaqueProofDigest(raw)
	if err != nil || hex.EncodeToString(got) != want {
		t.Fatal("persisted proof digest format changed")
	}
	for _, bad := range []string{raw + "=", raw[:42] + "9", raw[:5] + "\n" + raw[5:], raw[:42] + "\n"} {
		if ValidOpaqueProof(bad) {
			t.Fatal("noncanonical proof accepted")
		}
		if _, err := OpaqueProofDigest(bad); err == nil {
			t.Fatal("noncanonical proof digested")
		}
	}
	nonce, err := NewRandomToken()
	if err != nil || !ValidOpaqueProof(nonce) {
		t.Fatal("invalid random token", err)
	}
}
