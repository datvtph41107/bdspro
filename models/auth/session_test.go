package auth

import (
	"testing"
	"time"

	db "github.com/datvtph41107/bdspro/models/db"
)

func TestValidRefreshSession(t *testing.T) {
	now := time.Now()
	session := db.Session{AccountID: 7, RefreshToken: "token", ExpiresAt: now.Add(time.Minute)}
	if !ValidRefreshSession(session, 7, "token", now) {
		t.Fatal("valid session rejected")
	}
	session.IsBlocked = true
	if ValidRefreshSession(session, 7, "token", now) {
		t.Fatal("blocked session accepted")
	}
	session.IsBlocked = false
	if ValidRefreshSession(session, 8, "token", now) ||
		ValidRefreshSession(session, 7, "other", now) ||
		ValidRefreshSession(session, 7, "token", now.Add(2*time.Minute)) {
		t.Fatal("session authority/expiry check failed")
	}
}
