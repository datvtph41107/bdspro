package auth

import (
	"bytes"
	"testing"

	"github.com/google/uuid"
)

func TestPhoneNormalizationAndMAC(t *testing.T) {
	for _, raw := range []string{"0912345678", "+84912345678", "0912-345-678"} {
		got, err := NormalizePhone(raw, "VN")
		if err != nil || got != "+84912345678" {
			t.Fatal("normalization failed")
		}
	}
	for _, raw := range []string{"123", "+012345678", "+84912abc678", "+1234567890123456"} {
		if _, err := NormalizePhone(raw, "VN"); err == nil {
			t.Fatal("invalid phone accepted")
		}
	}
	code, err := NewPhoneCode()
	if err != nil || !ValidPhoneCode(code) {
		t.Fatal("invalid code")
	}
	id := uuid.New()
	browser := make([]byte, 32)
	mac := PhoneCodeMAC("test", id, "+84912345678", browser, code)
	if bytes.Equal(mac, PhoneCodeMAC("test", id, "+84912345679", browser, code)) ||
		bytes.Equal(mac, RegistrationCodeMAC("test", id, "+84912345678", browser, code)) {
		t.Fatal("code not bound to destination/purpose")
	}
}
