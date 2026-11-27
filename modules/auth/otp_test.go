package auth

import (
	"encoding/hex"
	"testing"

	"github.com/google/uuid"
)

func TestOTPWireCompatibility(t *testing.T) {
	const key = "disposable-otp-contract-key"
	id := uuid.MustParse("12345678-1234-1234-1234-123456789abc")
	session := uuid.MustParse("87654321-4321-4321-4321-cba987654321")
	browser := make([]byte, 32)
	for i := range browser {
		browser[i] = byte(i)
	}
	// Golden vectors computed independently using Python HMAC-SHA256 with
	// the original v1 byte transcript. These protect in-flight challenges.
	for _, tc := range []struct {
		name string
		got  []byte
		want string
	}{
		{"registration", RegistrationCodeMAC(key, id, "fixture@example.invalid", browser, "00001234"), "b1976154825dfe39e84b3e2926cbb1e30d1a332aed8cae49592a6cefe3f7847f"},
		{"phone", PhoneCodeMAC(key, id, "+84912345678", browser, "000123"), "ba800a96490f109344c10a571f96cfec6ad41f5c0c6178faddd3811e7c7b16a7"},
		{"email link", ContactLinkCodeMAC(key, id, 42, session, "email", "fixture@example.invalid", "00001234"), "f9c636ca705b566c7373f86716f0faaaf9c22a7b315b928d4ec1835f38387c58"},
		{"phone link", ContactLinkCodeMAC(key, id, 42, session, "phone", "+84912345678", "000123"), "edd8d38cdd01e15bc45c4c89478490b083cc91a06fbca423ea0fa2b18fc23078"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if hex.EncodeToString(tc.got) != tc.want {
				t.Fatal("persisted OTP MAC format changed")
			}
		})
	}
}

func TestOTPDecimalFormat(t *testing.T) {
	for _, tc := range []struct {
		name     string
		generate func() (string, error)
		valid    func(string) bool
		good     []string
		bad      []string
	}{
		{"email", NewRegistrationCode, ValidRegistrationCode, []string{"00000000", "99999999", "00001234"}, []string{"", "1234567", "123456789", "1234567a", "1234567 ", "１２３４", "123456\x00", "123456\xff"}},
		{"phone", NewPhoneCode, ValidPhoneCode, []string{"000000", "999999", "000123"}, []string{"", "12345", "1234567", "12345a", "12345 ", "１２", "12345\x00", "12345\xff"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, code := range tc.good {
				if !tc.valid(code) {
					t.Fatal("valid decimal code rejected")
				}
			}
			for _, code := range tc.bad {
				if tc.valid(code) {
					t.Fatal("invalid decimal code accepted")
				}
			}
			for i := 0; i < 32; i++ {
				code, err := tc.generate()
				if err != nil || !tc.valid(code) {
					t.Fatal("generator did not satisfy code format", err)
				}
			}
		})
	}
}
