package util

import (
	"fmt"
	"math/rand/v2"
	"strings"
)

const alphabet = "abcdefghijklmnopqrstuvwxyz"

// RandomInt returns a uniformly distributed integer in [min, max].
// It handles the full signed 64-bit range without overflowing the span.
func RandomInt(min, max int64) int64 {
	if min > max {
		panic("util.RandomInt: min exceeds max")
	}
	span := uint64(max) - uint64(min) + 1
	if span == 0 {
		return int64(rand.Uint64()) // Entire int64 domain.
	}
	return int64(uint64(min) + rand.Uint64N(span))
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
