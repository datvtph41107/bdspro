package util

import (
	"fmt"
	"math/rand/v2"
	"strings"
)

const alphabet = "abcdefghijklmnopqrstuvwxyz"

func RandomInt(min, max int64) int64 {
	if min > max {
		panic("util.RandomInt: min exceeds max")
	}
	span := uint64(max) - uint64(min) + 1
	if span == 0 {
		return int64(rand.Uint64())
	}
	return int64(uint64(min) + rand.Uint64N(span))
}

func RandomString(n int) string {
	var sb strings.Builder
	for i := 0; i < n; i++ {
		sb.WriteByte(alphabet[rand.IntN(len(alphabet))])
	}
	return sb.String()
}

func RandomPersonName() string {
	return RandomString(6)
}

func RandomOrganizationName() string {
	return RandomString(10)
}

func RandomEmail() string {
	return fmt.Sprintf("%s@example.test", RandomString(8))
}
