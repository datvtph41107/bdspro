package auth

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/datvtph41107/bdspro/modules/setting"
	"github.com/google/uuid"
)

func TestSymmetricKeyContract(t *testing.T) {
	id := uuid.New()
	key := strings.Repeat("a", setting.MinTokenSymmetricKeyBytes)
	short := key[:len(key)-1]
	nextKey := strings.Repeat("b", setting.MinTokenSymmetricKeyBytes)
	if _, err := NewJWTMaker(short); err == nil {
		t.Fatal("JWT accepted a short key")
	}
	if _, err := NewRefreshCredential(short, id, false); !errors.Is(err, ErrRefreshInvalid) {
		t.Fatal("refresh accepted a short issuance key")
	}
	maker, err := NewJWTMaker(key)
	if err != nil {
		t.Fatal(err)
	}
	raw, _, err := maker.CreateSessionAccessToken(1, time.Minute, id, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	nextMaker, err := NewJWTMaker(nextKey)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := nextMaker.VerifyAccessToken(raw); !errors.Is(err, ErrInvalidToken) {
		t.Fatal("old JWT accepted with new key")
	}
	refresh, err := NewRefreshCredential(key, id, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, wrong := range []string{short, nextKey} {
		if _, err := ReadRefreshCredential(wrong, refresh); !errors.Is(err, ErrRefreshInvalid) {
			t.Fatal("refresh accepted with invalid key")
		}
	}
	browser := make([]byte, 32)
	emailMAC := RegistrationCodeMAC(key, id, "fixture@example.invalid", browser, "00001234")
	phoneMAC := PhoneCodeMAC(key, id, "+84912345678", browser, "000123")
	linkMAC := ContactLinkCodeMAC(key, id, 1, id, "email", "fixture@example.invalid", "00001234")
	if bytes.Equal(emailMAC, RegistrationCodeMAC(nextKey, id, "fixture@example.invalid", browser, "00001234")) ||
		bytes.Equal(phoneMAC, PhoneCodeMAC(nextKey, id, "+84912345678", browser, "000123")) ||
		bytes.Equal(linkMAC, ContactLinkCodeMAC(nextKey, id, 1, id, "email", "fixture@example.invalid", "00001234")) {
		t.Fatal("OTP MAC unaffected by key change")
	}
	if bytes.Equal(emailMAC, phoneMAC) || bytes.Equal(emailMAC, linkMAC) || bytes.Equal(phoneMAC, linkMAC) {
		t.Fatal("OTP purpose MACs unexpectedly equal")
	}
}
