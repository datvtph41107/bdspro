package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"strconv"

	"github.com/google/uuid"
)

func ContactLinkCodeMAC(key string, id uuid.UUID, accountID int64, sessionID uuid.UUID, kind, destination, code string) []byte {
	mac := hmac.New(sha256.New, []byte(key))
	_, _ = mac.Write([]byte("bdspro.credential.link.v1\x00" + id.String() + "\x00" +
		strconv.FormatInt(accountID, 10) + "\x00" + sessionID.String() + "\x00" + kind + "\x00" + destination + "\x00" + code))
	return mac.Sum(nil)
}
