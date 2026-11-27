package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"math/big"
	"strconv"

	"github.com/google/uuid"
)

const (
	registrationCodeDigits = 8
	phoneCodeDigits        = 6
)

func NewRegistrationCode() (string, error) {
	return newDecimalCode(registrationCodeDigits)
}

func NewPhoneCode() (string, error) {
	return newDecimalCode(phoneCodeDigits)
}

// Both callers use a fixed, small digit count. Derive the exclusive random
// upper bound from that count so the generator and validator cannot drift.
func newDecimalCode(digits int) (string, error) {
	limit := int64(1)
	for i := 0; i < digits; i++ {
		limit *= 10
	}
	n, err := rand.Int(rand.Reader, big.NewInt(limit))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%0*d", digits, n.Int64()), nil
}

func ValidRegistrationCode(code string) bool {
	return validDecimalCode(code, registrationCodeDigits)
}

func ValidPhoneCode(code string) bool {
	return validDecimalCode(code, phoneCodeDigits)
}

func validDecimalCode(code string, digits int) bool {
	if len(code) != digits {
		return false
	}
	for i := 0; i < len(code); i++ {
		if code[i] < '0' || code[i] > '9' {
			return false
		}
	}
	return true
}

func RegistrationCodeMAC(key string, id uuid.UUID, email string, browser []byte, code string) []byte {
	return browserCodeMAC(key, "bdspro.email.registration.v1", id, email, browser, code)
}

func PhoneCodeMAC(key string, id uuid.UUID, phone string, browser []byte, code string) []byte {
	return browserCodeMAC(key, "bdspro.phone.continuation.v1", id, phone, browser, code)
}

// Preserve the v1 transcript exactly: NUL-delimited prefix, raw browser
// digest, then code. The two callers retain different purpose domains.
func browserCodeMAC(key, domain string, id uuid.UUID, destination string, browser []byte, code string) []byte {
	mac := hmac.New(sha256.New, []byte(key))
	_, _ = mac.Write([]byte(domain + "\x00" + id.String() + "\x00" + destination + "\x00"))
	_, _ = mac.Write(browser)
	_, _ = mac.Write([]byte(code))
	return mac.Sum(nil)
}

// Contact links bind an authenticated actor and session, not a browser.
func ContactLinkCodeMAC(key string, id uuid.UUID, accountID int64, sessionID uuid.UUID, kind, destination, code string) []byte {
	mac := hmac.New(sha256.New, []byte(key))
	_, _ = mac.Write([]byte("bdspro.credential.link.v1\x00" + id.String() + "\x00" +
		strconv.FormatInt(accountID, 10) + "\x00" + sessionID.String() + "\x00" + kind + "\x00" + destination + "\x00" + code))
	return mac.Sum(nil)
}
