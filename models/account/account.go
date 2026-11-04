// Package account owns persistent account creation and credentials.
// SQLC remains the query engine; models/db owns pgx transaction execution.
package account

import (
	"context"

	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/modules/util"
)

// Transactioner retains the model's existing public name; db owns the contract.
type Transactioner = db.Transactioner

type RegisterParams struct {
	Email        string
	PasswordHash string
}

// Register creates the Party, Person, Account, password credential and
// verification outbox row atomically; the caller controls HTTP error mapping.
func Register(ctx context.Context, tx Transactioner, arg RegisterParams) (db.Account, error) {
	var created db.Account
	err := tx.ExecTx(ctx, func(q *db.Queries) error {
		partyID, err := db.CreatePersonWithParty(ctx, q)
		if err != nil {
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
	if err != nil {
		if emailLoginTaken(err) {
			return db.Account{}, util.WithCause(ErrEmailRegistered, err)
		}
		return db.Account{}, err
	}
	return created, nil
}
