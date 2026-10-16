package util

import (
	"math/rand"
	"strings"
)

const alphabet = "abcdefghijklmnopqrstuvwxyz"

// RandomString supplies test fixtures only. Do not use math/rand for
// authentication secrets, session tokens or cryptographic keys.
func RandomString(n int) string {
	var sb strings.Builder
	for i := 0; i < n; i++ {
		sb.WriteByte(alphabet[rand.Intn(len(alphabet))])
	}
	return sb.String()
}

func RandomEmail() string {
	return RandomString(16) + "@example.test"
}
