package account

import (
	"time"

	db "github.com/datvtph41107/bdspro/models/db"
)

// Account is the canonical account model returned by model and service operations.
// PasswordResetVersion is persistence-only and never part of the API-facing model.
type Account struct {
	ID              int64
	PersonID        int64
	Email           string
	CreatedAt       time.Time
	IsEmailVerified bool
}

func AccountFromDB(row db.Account) *Account {
	return &Account{
		ID:              row.ID,
		PersonID:        row.PersonID,
		Email:           row.Email,
		CreatedAt:       row.CreatedAt,
		IsEmailVerified: row.IsEmailVerified,
	}
}
