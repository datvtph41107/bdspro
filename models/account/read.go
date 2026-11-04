package account

import (
	"context"
	"errors"

	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/jackc/pgx/v5"
)

// AccountReader is the existing persistence capability needed for account reads.
// A client token is not sufficient proof that its account still exists.
type AccountReader interface {
	GetAccountByID(context.Context, int64) (db.Account, error)
}

// GetByID returns the live account or the model-owned public account-unavailable
// failure. Unknown database errors stay private to the HTTP boundary.
func GetByID(ctx context.Context, reader AccountReader, id int64) (db.Account, error) {
	if id <= 0 {
		return db.Account{}, ErrAccountUnavailable
	}
	account, err := reader.GetAccountByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return db.Account{}, ErrAccountUnavailable
	}
	if err != nil {
		return db.Account{}, err
	}
	return account, nil
}
