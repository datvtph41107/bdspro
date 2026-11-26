package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"

	"github.com/google/uuid"
)

var ErrRefreshInvalid = errors.New("invalid refresh credential")

type RefreshProof struct {
	SessionID uuid.UUID
	Remember  bool
}

func refreshMAC(key, body string) []byte {
	derive := hmac.New(sha256.New, []byte(key))
	_, _ = derive.Write([]byte("bdspro.refresh.v1"))
	mac := hmac.New(sha256.New, derive.Sum(nil))
	_, _ = mac.Write([]byte(body))
	return mac.Sum(nil)
}

func NewRefreshCredential(key string, sessionID uuid.UUID, remember bool) (string, error) {
	if len(key) < 32 || sessionID == uuid.Nil {
		return "", ErrRefreshInvalid
	}
	nonce := make([]byte, 32)
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	flag := "0"
	if remember {
		flag = "1"
	}
	body := "r1." + sessionID.String() + "." + base64.RawURLEncoding.EncodeToString(nonce) + "." + flag
	return body + "." + base64.RawURLEncoding.EncodeToString(refreshMAC(key, body)), nil
}

func ReadRefreshCredential(key, raw string) (RefreshProof, error) {
	if len(key) < 32 || len(raw) > 256 {
		return RefreshProof{}, ErrRefreshInvalid
	}
	parts := strings.Split(raw, ".")
	if len(parts) != 5 || parts[0] != "r1" || (parts[3] != "0" && parts[3] != "1") {
		return RefreshProof{}, ErrRefreshInvalid
	}
	id, err := uuid.Parse(parts[1])
	if err != nil || id == uuid.Nil || id.String() != parts[1] {
		return RefreshProof{}, ErrRefreshInvalid
	}
	nonce, err := base64.RawURLEncoding.Strict().DecodeString(parts[2])
	if err != nil || len(nonce) != 32 {
		return RefreshProof{}, ErrRefreshInvalid
	}
	tag, err := base64.RawURLEncoding.Strict().DecodeString(parts[4])
	if err != nil || !hmac.Equal(tag, refreshMAC(key, strings.Join(parts[:4], "."))) {
		return RefreshProof{}, ErrRefreshInvalid
	}
	return RefreshProof{SessionID: id, Remember: parts[3] == "1"}, nil
}

func RefreshDigest(raw string) string {
	digest := sha256.Sum256([]byte(raw))
	return "sha256:" + hex.EncodeToString(digest[:])
}

func MatchesRefreshDigest(raw, stored string) bool {
	return subtle.ConstantTimeCompare([]byte(RefreshDigest(raw)), []byte(stored)) == 1
}
