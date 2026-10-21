package val

import (
	"errors"
	"net/mail"
	"strings"
)

var ErrInvalidEmail = errors.New("invalid email")

// Email returns the canonical login/invitation email representation.
// It rejects display names, whitespace within addresses and oversized data.
func Email(raw string) (string, error) {
	value := strings.ToLower(strings.TrimSpace(raw))
	parsed, err := mail.ParseAddress(value)
	if err != nil || parsed.Address != value || !strings.Contains(value, "@") ||
		len(value) > 254 || len(value) < 3 {
		return "", ErrInvalidEmail
	}
	return value, nil
}
