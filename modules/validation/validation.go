package validation

import (
	"net/mail"
	"strings"
	"unicode/utf8"
)

func PasswordLength(password string) bool {
	return utf8.RuneCountInString(password) >= 8 && len(password) <= 72
}

func EmailAddress(value string) (string, bool) {
	email := strings.ToLower(strings.TrimSpace(value))
	address, err := mail.ParseAddress(email)
	return email, err == nil && address.Address == email && len(email) <= 254
}
