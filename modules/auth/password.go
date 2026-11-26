package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"strings"
	"unicode/utf8"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrPasswordMismatch = errors.New("password does not match")
	ErrPasswordBusy     = errors.New("password verification capacity exhausted")
	ErrPasswordPolicy   = errors.New("password does not meet length policy")
)

const (
	argonMemory = 19 * 1024
	argonTime   = 2
	argonLanes  = 1
	argonPrefix = "$argon2id$v=19$m=19456,t=2,p=1$"
)

var passwordSlots = make(chan struct{}, 4)

func ValidateNewPassword(password string) error {
	if !utf8.ValidString(password) || len(password) > 512 {
		return ErrPasswordPolicy
	}
	n := utf8.RuneCountInString(password)
	if n < 15 || n > 128 {
		return ErrPasswordPolicy
	}
	return nil
}

func acquirePasswordSlot() bool {
	select {
	case passwordSlots <- struct{}{}:
		return true
	default:
		return false
	}
}

func HashPassword(password string) (string, error) {
	if err := ValidateNewPassword(password); err != nil {
		return "", err
	}
	if !acquirePasswordSlot() {
		return "", ErrPasswordBusy
	}
	defer func() { <-passwordSlots }()
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	digest := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonLanes, 32)
	return argonPrefix + base64.RawStdEncoding.EncodeToString(salt) + "$" + base64.RawStdEncoding.EncodeToString(digest), nil
}

func CheckPassword(password string, stored string) error {
	if !utf8.ValidString(password) || len(password) > 512 || len(stored) > 256 {
		return ErrPasswordMismatch
	}
	if !acquirePasswordSlot() {
		return ErrPasswordBusy
	}
	defer func() { <-passwordSlots }()
	if strings.HasPrefix(stored, argonPrefix) {
		parts := strings.Split(strings.TrimPrefix(stored, argonPrefix), "$")
		if len(parts) != 2 {
			return ErrPasswordMismatch
		}
		salt, err := base64.RawStdEncoding.Strict().DecodeString(parts[0])
		if err != nil || len(salt) != 16 {
			return ErrPasswordMismatch
		}
		expected, err := base64.RawStdEncoding.Strict().DecodeString(parts[1])
		if err != nil || len(expected) != 32 {
			return ErrPasswordMismatch
		}
		got := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonLanes, 32)
		if subtle.ConstantTimeCompare(got, expected) == 1 {
			return nil
		}
		return ErrPasswordMismatch
	}
	if strings.HasPrefix(stored, "$2a$") || strings.HasPrefix(stored, "$2b$") || strings.HasPrefix(stored, "$2y$") {
		cost, err := bcrypt.Cost([]byte(stored))
		if err != nil || cost < bcrypt.MinCost || cost > 14 || len(password) > 72 {
			return ErrPasswordMismatch
		}
		if bcrypt.CompareHashAndPassword([]byte(stored), []byte(password)) == nil {
			return nil
		}
	}
	return ErrPasswordMismatch
}

func CheckMissingPassword(password string) error {
	dummy := argonPrefix + "AAAAAAAAAAAAAAAAAAAAAA$AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
	if err := CheckPassword(password, dummy); errors.Is(err, ErrPasswordBusy) {
		return err
	}
	return ErrPasswordMismatch
}
