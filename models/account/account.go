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
	created, err := db.WithTxResult(ctx, tx, func(q *db.Queries) (db.Account, error) {
		partyID, err := db.CreatePersonWithParty(ctx, q)
		if err != nil {
			return db.Account{}, err
		}
		account, err := q.CreateAccount(ctx, db.CreateAccountParams{
			PersonID: partyID, Email: arg.Email,
		})
		if err != nil {
			return db.Account{}, err
		}
		if err := q.CreateAccountSettings(ctx, account.ID); err != nil {
			return db.Account{}, err
		}
		if err := q.CreatePasswordCredential(ctx, db.CreatePasswordCredentialParams{
			AccountID: account.ID, PasswordHash: arg.PasswordHash,
		}); err != nil {
			return db.Account{}, err
		}
		return account, q.RecordEmailVerificationJob(ctx, account.ID)
	})
	if emailLoginTaken(err) {
		return db.Account{}, util.WithCause(ErrEmailRegistered, err)
	}
	return created, err
}
