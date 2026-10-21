package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
)

func randomToken() (string, string, error) {
	var data [32]byte
	if _, err := rand.Read(data[:]); err != nil {
		return "", "", err
	}
	token := base64.RawURLEncoding.EncodeToString(data[:])
	digest := sha256.Sum256([]byte(token))
	return token, hex.EncodeToString(digest[:]), nil
}
func digestToken(token string) string {
	digest := sha256.Sum256([]byte(token))
	return hex.EncodeToString(digest[:])
}
