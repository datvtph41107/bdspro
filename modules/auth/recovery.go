package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
)

var ErrRecoveryProof = errors.New("invalid password recovery proof")

func NewRecoveryProof() (string, []byte, error) {
	var secret [32]byte
	if _, err := rand.Read(secret[:]); err != nil {
		return "", nil, err
	}
	raw := base64.RawURLEncoding.EncodeToString(secret[:])
	digest, err := RecoveryDigest(raw)
	return raw, digest, err
}

func RecoveryDigest(raw string) ([]byte, error) {
	if len(raw) != 43 {
		return nil, ErrRecoveryProof
	}
	secret, err := base64.RawURLEncoding.Strict().DecodeString(raw)
	if err != nil || len(secret) != 32 {
		return nil, ErrRecoveryProof
	}
	digest := sha256.Sum256([]byte(raw))
	return digest[:], nil
}
