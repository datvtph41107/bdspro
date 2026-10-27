package util

import (
	cryptorand "crypto/rand"
	"encoding/hex"
	"fmt"
	"math/rand/v2"
	"strings"
)

const alphabet = "abcdefghijklmnopqrstuvwxyz"

// RandomInt generates a random integer between min and max, inclusive.
func RandomInt(min, max int64) int64 {
	return min + rand.Int64N(max-min+1)
}

// RandomString generates a random lowercase string of length n.
func RandomString(n int) string {
	var sb strings.Builder
	for i := 0; i < n; i++ {
		sb.WriteByte(alphabet[rand.IntN(len(alphabet))])
	}
	return sb.String()
}

// RandomPersonName generates sample person names for development and tests.
func RandomPersonName() string {
	return RandomString(6)
}

// RandomOrganizationName generates sample organization names for development and tests.
func RandomOrganizationName() string {
	return RandomString(10)
}

// RandomEmail generates a sample email address.
func RandomEmail() string {
	return fmt.Sprintf("%s@example.test", RandomString(8))
}

// RandomSecret generates a cryptographically secure one-time email token.
// Do not use math/rand for authentication codes.
func RandomSecret() (string, error) {
	bytes := make([]byte, 32)
	if _, err := cryptorand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
