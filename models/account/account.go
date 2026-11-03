// Package account owns persistent account creation and credentials.
// SQLC remains the query engine; models/db owns pgx transaction execution.
package account

import (
	"context"

	db "github.com/datvtph41107/bdspro/models/db"
)

type Transactioner interface {
	ExecTx(context.Context, func(*db.Queries) error) error
}

type RegisterParams struct {
	Email        string
	PasswordHash string
}

// Register creates the Party, Person, Account, password credential and
// verification outbox row atomically; the caller controls HTTP error mapping.
func Register(ctx context.Context, tx Transactioner, arg RegisterParams) (db.Account, error) {
	var created db.Account
	err := tx.ExecTx(ctx, func(q *db.Queries) error {
		partyID, err := q.CreateParty(ctx)
		if err != nil {
			return err
		}
		if err := q.CreatePerson(ctx, partyID); err != nil {
			return err
		}
		created, err = q.CreateAccount(ctx, db.CreateAccountParams{
			PersonID: partyID, Email: arg.Email,
		})
		if err != nil {
			return err
		}
		if err := q.CreatePasswordCredential(ctx, db.CreatePasswordCredentialParams{
			AccountID: created.ID, PasswordHash: arg.PasswordHash,
		}); err != nil {
			return err
		}
		return q.RecordEmailVerificationJob(ctx, created.ID)
	})
	return created, err
}
