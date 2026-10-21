package util

import (
	"errors"
	"strings"
)

// ErrInvalidDecimal reports a decimal that is not positive or exceeds the
// agreed integer/fraction precision. Never convert exact money to float64.
var ErrInvalidDecimal = errors.New("invalid positive decimal")

// DecimalString validates a positive base-10 decimal and returns the stable
// canonical representation used for request hashing and decimal SQL columns.
// It rejects exponents, signs, NaN, spaces inside values and zero.
func DecimalString(raw string, integerDigits, fractionDigits int) (string, error) {
	value := strings.TrimSpace(raw)
	if integerDigits < 1 || fractionDigits < 0 || value == "" {
		return "", ErrInvalidDecimal
	}
	parts := strings.Split(value, ".")
	if len(parts) < 1 || len(parts) > 2 || len(parts[0]) == 0 ||
		len(parts[0]) > integerDigits {
		return "", ErrInvalidDecimal
	}
	if len(parts) == 2 && (len(parts[1]) == 0 || len(parts[1]) > fractionDigits) {
		return "", ErrInvalidDecimal
	}
	for _, part := range parts {
		for i := 0; i < len(part); i++ {
			if part[i] < '0' || part[i] > '9' {
				return "", ErrInvalidDecimal
			}
		}
	}
	whole := strings.TrimLeft(parts[0], "0")
	if whole == "" {
		whole = "0"
	}
	if len(parts) == 1 {
		if whole == "0" {
			return "", ErrInvalidDecimal
		}
		return whole, nil
	}
	fraction := strings.TrimRight(parts[1], "0")
	if whole == "0" && fraction == "" {
		return "", ErrInvalidDecimal
	}
	if fraction == "" {
		return whole, nil
	}
	return whole + "." + fraction, nil
}
