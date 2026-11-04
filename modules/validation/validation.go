// Package validation contains reusable input rules without transport dependencies.
package validation

import (
	"net/mail"
	"strings"
	"unicode/utf8"
)

// PasswordLength checks the application's password-length limits.
// The 72-byte upper bound is necessary for bcrypt.
func PasswordLength(password string) bool {
	return utf8.RuneCountInString(password) >= 8 && len(password) <= 72
}

// EmailAddress canonicalizes a mailbox used for login and verification.
// It does not decide whether an address is already linked to an account.
func EmailAddress(value string) (string, bool) {
	email := strings.ToLower(strings.TrimSpace(value))
	address, err := mail.ParseAddress(email)
	return email, err == nil && address.Address == email && len(email) <= 254
}
