package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"

	"github.com/datvtph41107/bdspro/modules/setting"
	"github.com/google/uuid"
)

const refreshVersion = "r1"

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
	if len(key) < setting.MinTokenSymmetricKeyBytes || sessionID == uuid.Nil {
		return "", ErrRefreshInvalid
	}
	nonce, err := NewRandomToken()
	if err != nil {
		return "", err
	}
	flag := "0"
	if remember {
		flag = "1"
	}
	body := refreshVersion + "." + sessionID.String() + "." + nonce + "." + flag
	return body + "." + base64.RawURLEncoding.EncodeToString(refreshMAC(key, body)), nil
}

func ReadRefreshCredential(key, raw string) (RefreshProof, error) {
	if len(key) < setting.MinTokenSymmetricKeyBytes || len(raw) > 256 {
		return RefreshProof{}, ErrRefreshInvalid
	}
	parts := strings.Split(raw, ".")
	if len(parts) != 5 || parts[0] != refreshVersion || (parts[3] != "0" && parts[3] != "1") {
		return RefreshProof{}, ErrRefreshInvalid
	}
	id, err := uuid.Parse(parts[1])
	if err != nil || id == uuid.Nil || id.String() != parts[1] {
		return RefreshProof{}, ErrRefreshInvalid
	}
	if !ValidOpaqueProof(parts[2]) {
		return RefreshProof{}, ErrRefreshInvalid
	}
	// Compare the canonical encoded tag, not decoded bytes. Base64 decoders
	// ignore CR/LF; aliases must not reach the persisted refresh-digest check.
	expected := base64.RawURLEncoding.EncodeToString(refreshMAC(key, strings.Join(parts[:4], ".")))
	if !hmac.Equal([]byte(parts[4]), []byte(expected)) {
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
