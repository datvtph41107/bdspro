package auth

import (
	"bytes"
	"testing"

	"github.com/google/uuid"
)

func TestRegistrationCodeBinding(t *testing.T) {
	code, err := NewRegistrationCode()
	if err != nil || !ValidRegistrationCode(code) {
		t.Fatal("invalid generated code")
	}
	id := uuid.New()
	key := "disposable-test-secret"
	browser := make([]byte, 32)
	mac := RegistrationCodeMAC(key, id, "user@example.invalid", browser, code)
	if !bytes.Equal(mac, RegistrationCodeMAC(key, id, "user@example.invalid", browser, code)) {
		t.Fatal("unstable code MAC")
	}
	browser[0] = 1
	if bytes.Equal(mac, RegistrationCodeMAC(key, id, "user@example.invalid", browser, code)) ||
		bytes.Equal(mac, RegistrationCodeMAC(key, uuid.New(), "user@example.invalid", make([]byte, 32), code)) {
		t.Fatal("code not bound to browser and challenge")
	}
}
