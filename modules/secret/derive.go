package secret

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
)

// ValidDerivationKey is the token-derivation key requirement shared by
// preflight validation and the derivation itself. This rejects weak keys
// without computing a disposable HMAC or opening a database transaction.
func ValidDerivationKey(signingKey string) bool {
	return len(signingKey) >= 32
}

// DeriveToken creates a stable 256-bit secret for an opaque random ID.
// Domain separation prevents tokens from being reused across workflows.
// Keep old signing keys available until their outstanding proofs expire.
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
