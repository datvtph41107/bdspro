package secret

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
)

func ValidDerivationKey(signingKey string) bool {
	return len(signingKey) >= 32
}

func DeriveToken(signingKey, purpose string, id []byte) (string, error) {
	if !ValidDerivationKey(signingKey) || purpose == "" || len(id) < 16 {
		return "", errors.New("invalid token derivation parameters")
	}
	mac := hmac.New(sha256.New, []byte(signingKey))
	_, _ = mac.Write([]byte(purpose))
	_, _ = mac.Write([]byte{0})
	_, _ = mac.Write(id)
	return hex.EncodeToString(mac.Sum(nil)), nil
}
