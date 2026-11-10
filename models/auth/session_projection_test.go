package auth

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/google/uuid"
)

func TestSessionModelNeverExposesRefreshCredential(t *testing.T) {
	now := time.Date(2026, 11, 10, 9, 0, 0, 0, time.UTC)
	row := db.Session{
		ID: uuid.New(), AccountID: 12, UserAgent: "browser",
		ClientIp: "198.51.100.2", RefreshToken: "sha256:VERY_PRIVATE_FINGERPRINT",
		CreatedAt: now, ExpiresAt: now.Add(time.Hour),
	}
	model := SessionFromDB(row)
	if model.ID != row.ID || model.AccountID != row.AccountID ||
		model.UserAgent != row.UserAgent || model.ClientIP != row.ClientIp ||
		model.ExpiresAt != row.ExpiresAt {
		t.Fatalf("session metadata projection changed: %+v", model)
	}
	encoded, err := json.Marshal(model)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), row.RefreshToken) ||
		strings.Contains(string(encoded), "RefreshToken") {
		t.Fatalf("session model exposed refresh authority: %s", encoded)
	}
	row.UserAgent = "changed"
	if model.UserAgent != "browser" {
		t.Fatal("session metadata unexpectedly aliases persistence row")
	}
}

func TestSessionInfoProjectionPreservesPublicFields(t *testing.T) {
	now := time.Date(2026, 11, 10, 9, 0, 0, 0, time.UTC)
	id := uuid.New()
	row := db.ListActiveSessionsRow{
		ID: id, UserAgent: "browser", ClientIp: "203.0.113.5",
		CreatedAt: now, ExpiresAt: now.Add(time.Hour),
	}
	if sessionInfosFromDB(nil) != nil {
		t.Fatal("nil SQLC rows must remain nil at model boundary")
	}
	empty := sessionInfosFromDB([]db.ListActiveSessionsRow{})
	if empty == nil || len(empty) != 0 {
		t.Fatalf("allocated empty session list changed: %+v", empty)
	}
	got := sessionInfosFromDB([]db.ListActiveSessionsRow{row})
	if len(got) != 1 || got[0].ID != row.ID ||
		got[0].UserAgent != row.UserAgent || got[0].ClientIP != row.ClientIp ||
		got[0].CreatedAt != row.CreatedAt || got[0].ExpiresAt != row.ExpiresAt {
		t.Fatalf("session-list projection changed: %+v", got)
	}
}
