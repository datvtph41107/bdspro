package account

import (
	"context"
	"errors"

	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/jackc/pgx/v5"
)

// ReadByID resolves account persistence from the context-owned SQLC executor.
// Only this model interprets an absent account as ErrAccountUnavailable.
func ReadByID(ctx context.Context, id int64) (db.Account, error) {
	if id <= 0 {
		return db.Account{}, ErrAccountUnavailable
	}
	q, err := db.GetQueries(ctx)
	if err != nil {
		return db.Account{}, err
	}
	row, err := q.ReadAccountByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return db.Account{}, ErrAccountUnavailable
	}
	if err != nil {
		return db.Account{}, err
	}
	return db.Account{
		ID: row.ID, PersonID: row.PersonID, Email: row.Email,
		CreatedAt: row.CreatedAt, IsEmailVerified: row.IsEmailVerified,
	}, nil
}

// ReadByEmail keeps account-missing semantics at the account model boundary.
// PostgreSQL/SQLC errors are not globally rewritten or mapped by models/db.
func ReadByEmail(ctx context.Context, email string) (db.Account, error) {
	q, err := db.GetQueries(ctx)
	if err != nil {
		return db.Account{}, err
	}
	row, err := q.ReadAccountByEmail(ctx, email)
	if err != nil {
		return db.Account{}, err
	}
	return db.Account{
		ID: row.ID, PersonID: row.PersonID, Email: row.Email,
		CreatedAt: row.CreatedAt, IsEmailVerified: row.IsEmailVerified,
	}, nil
}
