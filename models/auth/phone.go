package auth

import (
	"context"

	db "github.com/datvtph41107/bdspro/models/db"
)

// PhoneOperations owns phone credential and OTP persistence behavior.
// The database package only supplies SQLC queries and pgx transactions.
type PhoneOperations interface {
	AccountIDForPhone(context.Context, string) (int64, error)
	StartChallenge(context.Context, string, string, int64, []byte, string) error
	ConsumeChallenge(context.Context, string, string, int64, []byte) error
	StartPhoneSignup(context.Context, string, []byte, string) error
	CompletePhoneSignup(context.Context, string, []byte) (db.Account, error)
}

type PhoneStore interface {
	db.Store
	PhoneOperations
}

type phoneStore struct {
	db.Store
}

// NewPhoneStore adapts the existing Store without creating another connection,
// query implementation, or database pool.
func NewPhoneStore(store db.Store) PhoneStore {
	return &phoneStore{Store: store}
}

var _ PhoneStore = (*phoneStore)(nil)
