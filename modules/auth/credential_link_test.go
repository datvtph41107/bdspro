package auth

import (
	"bytes"
	"testing"

	"github.com/google/uuid"
)

func TestContactLinkCodeBinding(t *testing.T) {
	id, session := uuid.New(), uuid.New()
	const key, email, code = "test-only-secret", "user@example.invalid", "12345678"
	base := ContactLinkCodeMAC(key, id, 42, session, "email", email, code)
	for _, other := range [][]byte{
		ContactLinkCodeMAC(key, id, 43, session, "email", email, code),
		ContactLinkCodeMAC(key, id, 42, uuid.New(), "email", email, code),
		ContactLinkCodeMAC(key, uuid.New(), 42, session, "email", email, code),
		ContactLinkCodeMAC(key, id, 42, session, "phone", email, code),
		ContactLinkCodeMAC(key, id, 42, session, "email", "other@example.invalid", code),
		ContactLinkCodeMAC(key, id, 42, session, "email", email, "87654321"),
		RegistrationCodeMAC(key, id, email, make([]byte, 32), code),
	} {
		if bytes.Equal(base, other) {
			t.Fatal("contact proof is not fully scoped")
		}
	}
}
