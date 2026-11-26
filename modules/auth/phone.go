package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

var ErrPhoneFormat = errors.New("invalid phone number")
var e164 = regexp.MustCompile(`^\+[1-9][0-9]{7,14}$`)

func NormalizePhone(raw, country string) (string, error) {
	phone := strings.TrimSpace(raw)
	phone = strings.ReplaceAll(phone, " ", "")
	phone = strings.ReplaceAll(phone, "-", "")
	if !strings.HasPrefix(phone, "+") && strings.EqualFold(country, "VN") && len(phone) == 10 && strings.HasPrefix(phone, "0") {
		phone = "+84" + phone[1:]
	}
	if !e164.MatchString(phone) {
		return "", ErrPhoneFormat
	}
	return phone, nil
}

func NewPhoneCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

func ValidPhoneCode(code string) bool {
	return len(code) == 6 && strings.IndexFunc(code, func(r rune) bool { return r < '0' || r > '9' }) < 0
}

func PhoneCodeMAC(key string, id uuid.UUID, phone string, browser []byte, code string) []byte {
	mac := hmac.New(sha256.New, []byte(key))
	_, _ = mac.Write([]byte("bdspro.phone.continuation.v1\x00" + id.String() + "\x00" + phone + "\x00"))
	_, _ = mac.Write(browser)
	_, _ = mac.Write([]byte(code))
	return mac.Sum(nil)
}
