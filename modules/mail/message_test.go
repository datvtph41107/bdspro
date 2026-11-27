package mail

import (
	"strings"
	"testing"
)

func TestPlainMessage(t *testing.T) {
	proof := strings.Repeat("A", 43)
	m, err := plainMessage("security@example.invalid", "user@example.invalid", "BDSPro password recovery", proof)
	if err != nil || !strings.Contains(m, proof) {
		t.Fatal("expected valid plain message")
	}
	if _, err := plainMessage("security@example.invalid", "user@example.invalid\r\nBcc: other@example.invalid", "BDSPro password recovery", proof); err == nil {
		t.Fatal("header injection accepted")
	}
}
