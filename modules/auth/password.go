package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
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
	argonMemory                  = 19 * 1024
	argonTime                    = 2
	argonLanes                   = 1
	argonSaltBytes               = 16
	argonDigestBytes             = 32
	passwordMaxBytes             = 512
	passwordMinRunes             = 15
	passwordMaxRunes             = 128
	passwordMaxHashBytes         = 256
	legacyBcryptMaxCost          = 14
	legacyBcryptMaxPasswordBytes = 72
	passwordConcurrency          = 4
)

// The PHC header must describe the exact parameters used to derive the digest.
// Changing the accepted format requires explicit legacy verification support.
var argonPrefix = fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$", argon2.Version, argonMemory, argonTime, argonLanes)

var missingPasswordHash = argonPrefix + base64.RawStdEncoding.EncodeToString(make([]byte, argonSaltBytes)) + "$" +
	base64.RawStdEncoding.EncodeToString(make([]byte, argonDigestBytes))

var passwordSlots = make(chan struct{}, passwordConcurrency)

func validateNewPassword(password string) error {
	if !utf8.ValidString(password) || len(password) > passwordMaxBytes {
		return ErrPasswordPolicy
	}
	n := utf8.RuneCountInString(password)
	if n < passwordMinRunes || n > passwordMaxRunes {
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
	if err := validateNewPassword(password); err != nil {
		return "", err
	}
	if !acquirePasswordSlot() {
		return "", ErrPasswordBusy
	}
	defer func() { <-passwordSlots }()
	salt := make([]byte, argonSaltBytes)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	digest := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonLanes, argonDigestBytes)
	return argonPrefix + base64.RawStdEncoding.EncodeToString(salt) + "$" + base64.RawStdEncoding.EncodeToString(digest), nil
}

func CheckPassword(password string, stored string) error {
	if !utf8.ValidString(password) || len(password) > passwordMaxBytes || len(stored) > passwordMaxHashBytes {
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
		if err != nil || len(salt) != argonSaltBytes {
			return ErrPasswordMismatch
		}
		expected, err := base64.RawStdEncoding.Strict().DecodeString(parts[1])
		if err != nil || len(expected) != argonDigestBytes {
			return ErrPasswordMismatch
		}
		got := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonLanes, argonDigestBytes)
		if subtle.ConstantTimeCompare(got, expected) == 1 {
			return nil
		}
		return ErrPasswordMismatch
	}
	if strings.HasPrefix(stored, "$2a$") || strings.HasPrefix(stored, "$2b$") || strings.HasPrefix(stored, "$2y$") {
		cost, err := bcrypt.Cost([]byte(stored))
		if err != nil || cost < bcrypt.MinCost || cost > legacyBcryptMaxCost || len(password) > legacyBcryptMaxPasswordBytes {
			return ErrPasswordMismatch
		}
		if bcrypt.CompareHashAndPassword([]byte(stored), []byte(password)) == nil {
			return nil
		}
	}
	return ErrPasswordMismatch
}

func CheckMissingPassword(password string) error {
	if err := CheckPassword(password, missingPasswordHash); errors.Is(err, ErrPasswordBusy) {
		return err
	}
	return ErrPasswordMismatch
}
