package util

import (
	"errors"
	"strings"
)

var ErrInvalidCurrency = errors.New("invalid currency code")

// CurrencyCode returns uppercase three-letter ASCII currency notation.
// This is shape validation, not a claim that a code is in a live ISO registry.
func CurrencyCode(raw string) (string, error) {
	code := strings.ToUpper(strings.TrimSpace(raw))
	if len(code) != 3 {
		return "", ErrInvalidCurrency
	}
	for _, c := range code {
		if c < 'A' || c > 'Z' {
			return "", ErrInvalidCurrency
		}
	}
	return code, nil
}
