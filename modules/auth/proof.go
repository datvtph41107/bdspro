package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
)

// PasswordResetTTLMinutes applies to password recovery issuance, including the legacy job.
const PasswordResetTTLMinutes = 15

const (
	opaqueSecretBytes = 32
	opaqueTokenLength = (opaqueSecretBytes*8 + 5) / 6
)

var ErrOpaqueProof = errors.New("invalid opaque proof")

// NewRandomToken returns 256 random bits as canonical unpadded base64url.
func NewRandomToken() (string, error) {
	var secret [opaqueSecretBytes]byte
	if _, err := rand.Read(secret[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(secret[:]), nil
}

func NewOpaqueProof() (string, []byte, error) {
	raw, err := NewRandomToken()
	if err != nil {
		return "", nil, err
	}
	digest := sha256.Sum256([]byte(raw))
	return raw, digest[:], nil
}

// ValidOpaqueProof checks only the encoding, not possession or authorization.
func ValidOpaqueProof(raw string) bool {
	if len(raw) != opaqueTokenLength {
		return false
	}
	secret, err := base64.RawURLEncoding.Strict().DecodeString(raw)
	return err == nil && len(secret) == opaqueSecretBytes
}

func OpaqueProofDigest(raw string) ([]byte, error) {
	if !ValidOpaqueProof(raw) {
		return nil, ErrOpaqueProof
	}
	// Persist the SHA-256 of the encoded string, not of the decoded secret.
	digest := sha256.Sum256([]byte(raw))
	return digest[:], nil
}
