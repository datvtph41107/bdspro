package convert

import (
	"testing"
	"time"

	account_model "github.com/datvtph41107/bdspro/models/account"
)

func TestToUserReturnsPersistedAccountPersonRelation(t *testing.T) {
	created := time.Date(2026, 11, 5, 0, 0, 0, 0, time.UTC)
	account := account_model.Account{
		ID: 15, PersonID: 27, Email: "user@example.test",
		CreatedAt: created, IsEmailVerified: true,
	}
	user := ToUser(&account)
	if user.AccountID != account.ID || user.PersonID != account.PersonID ||
		user.Email != account.Email || user.IsEmailVerified != account.IsEmailVerified ||
		!user.CreatedAt.Equal(created) {
		t.Fatalf("user projection lost persisted identity: %+v", user)
	}
}
