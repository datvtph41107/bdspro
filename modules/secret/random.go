package secret

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

func RandomHex(n int) (string, error) {
	if n <= 0 {
		return "", fmt.Errorf("random byte count must be positive")
	}
	data := make([]byte, n)
	if _, err := rand.Read(data); err != nil {
		return "", err
	}
	return hex.EncodeToString(data), nil
}
