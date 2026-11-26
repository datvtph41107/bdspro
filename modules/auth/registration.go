package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"math/big"

	"github.com/google/uuid"
)

func NewRegistrationCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(100000000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%08d", n.Int64()), nil
}

func ValidRegistrationCode(code string) bool {
	if len(code) != 8 {
		return false
	}
	for _, c := range code {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func RegistrationCodeMAC(key string, id uuid.UUID, email string, browser []byte, code string) []byte {
	mac := hmac.New(sha256.New, []byte(key))
	_, _ = mac.Write([]byte("bdspro.email.registration.v1\x00" + id.String() + "\x00" + email + "\x00"))
	_, _ = mac.Write(browser)
	_, _ = mac.Write([]byte(code))
	return mac.Sum(nil)
}
