package account

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/google/uuid"
)

func TestAccountProjectionExcludesPasswordRecoveryVersion(t *testing.T) {
	now := time.Date(2026, 11, 10, 9, 0, 0, 0, time.UTC)
	row := db.Account{
		ID: 17, PersonID: 19, Email: "person@example.test",
		CreatedAt: now, IsEmailVerified: true, PasswordResetVersion: 43,
	}
	account := AccountFromDB(row)
	if account.ID != row.ID || account.PersonID != row.PersonID ||
		account.Email != row.Email || account.CreatedAt != now ||
		!account.IsEmailVerified {
		t.Fatalf("account model lost safe metadata: %+v", account)
	}
	encoded, err := json.Marshal(account)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "PasswordResetVersion") ||
		strings.Contains(string(encoded), "password_reset_version") {
		t.Fatalf("account model exposed recovery metadata: %s", encoded)
	}
	row.Email = "changed@example.test"
	if account.Email != "person@example.test" {
		t.Fatal("account model unexpectedly aliases SQLC storage")
	}
}

func TestAccountConnectionProjectionPreservesNilAndOwnerBoundary(t *testing.T) {
	now := time.Date(2026, 11, 10, 9, 0, 0, 0, time.UTC)
	id := uuid.New()
	row := db.ListAccountConnectionHistoriesRow{
		ID: 12, SessionID: id,
		Action: "login", IpAddress: "203.0.113.3",
		UserAgent: "test", CreatedAt: now,
	}
	if connectionsFromDB(nil) != nil {
		t.Fatal("nil database result must remain nil at model boundary")
	}
	empty := connectionsFromDB([]db.ListAccountConnectionHistoriesRow{})
	if empty == nil || len(empty) != 0 {
		t.Fatalf("allocated empty connection result changed: %+v", empty)
	}
	got := connectionsFromDB([]db.ListAccountConnectionHistoriesRow{row})
	if len(got) != 1 || got[0].ID != row.ID ||
		got[0].SessionID != row.SessionID || got[0].IPAddress != row.IpAddress ||
		got[0].CreatedAt != now {
		t.Fatalf("connection projection changed: %+v", got)
	}
	encoded, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "AccountID") {
		t.Fatalf("connection model included owner identifier: %s", encoded)
	}
}
