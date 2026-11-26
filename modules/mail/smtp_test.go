package mail

import (
	"strings"
	"testing"
)

func TestResetMessage(t *testing.T) {
	proof := strings.Repeat("A", 43)
	m, err := resetMessage("security@example.invalid", "user@example.invalid", proof)
	if err != nil || !strings.Contains(m, proof) {
		t.Fatal("expected valid recovery message")
	}
	if _, err := resetMessage("security@example.invalid", "user@example.invalid\r\nBcc: other@example.invalid", proof); err == nil {
		t.Fatal("header injection accepted")
	}
}
