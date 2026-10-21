package util

import (
	"errors"
	"strings"
	"time"
)

var ErrInvalidTimestamp = errors.New("invalid RFC3339 timestamp")

// TimestampUTC accepts an optional RFC3339 timestamp and formats it as
// canonical UTC RFC3339Nano text for stable idempotency hashing and SQL input.
// Empty input stays empty; it does not silently become the current time.
func TimestampUTC(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", nil
	}
	t, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return "", ErrInvalidTimestamp
	}
	return t.UTC().Format(time.RFC3339Nano), nil
}
