package account

import (
	"context"
	"errors"

	auth_model "github.com/datvtph41107/bdspro/models/auth"
	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/modules/util"
	"github.com/datvtph41107/bdspro/modules/validation"
	"github.com/jackc/pgx/v5"
)

// Transactioner retains the model's existing public name; db owns the contract.
type Transactioner = db.Transactioner

type RegisterParams struct {
	Email        string
	PasswordHash string
}

// Register creates the Party, Person, Account, password credential and
// verification outbox row atomically; the caller controls HTTP error mapping.
func Register(ctx context.Context, tx Transactioner, arg RegisterParams) (*Account, error) {
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
		return nil, util.WithCause(ErrEmailRegistered, err)
	}
	if err != nil {
		return nil, err
	}
	return AccountFromDB(created), nil
}

// AccountReader is the existing persistence capability needed for account reads.
// A client token is not sufficient proof that its account still exists.
type AccountReader interface {
	GetAccountByID(context.Context, int64) (db.Account, error)
}

// GetByID returns the live account or the model-owned public account-unavailable
// failure. Unknown database errors stay private to the HTTP boundary.
func GetByID(ctx context.Context, reader AccountReader, id int64) (*Account, error) {
	if id <= 0 {
		return nil, ErrAccountUnavailable
	}
	account, err := reader.GetAccountByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrAccountUnavailable
	}
	if err != nil {
		return nil, err
	}
	return AccountFromDB(account), nil
}

// AccountEmailReader is the account model's one email lookup capability.
// The model—not AuthService—normalizes the missing-account condition.
type AccountEmailReader interface {
	GetAccountByEmail(context.Context, string) (db.Account, error)
}

// GetByEmail maps an existing account into the same canonical representation
// used for registration, OIDC registration and ID-based reads.
func GetByEmail(ctx context.Context, reader AccountEmailReader, email string) (*Account, error) {
	row, err := reader.GetAccountByEmail(ctx, email)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrAccountUnavailable
	}
	if err != nil {
		return nil, err
	}
	return AccountFromDB(row), nil
}

// CreateOAuth2Account atomically creates a passwordless account and its OIDC
// identity. This must ONLY be called after verifying an ID token and a verified
// email from a trusted issuer. It never links an existing account by email.
func CreateOAuth2Account(ctx context.Context, tx Transactioner, identity auth_model.OIDCClaims) (*Account, error) {
	key, err := identity.Key()
	if err != nil {
		return nil, err
	}
	email, valid := validation.EmailAddress(identity.Email)
	if !identity.EmailVerified || !valid {
		return nil, auth_model.ErrInvalidEmail
	}
	created, err := db.WithTxResult(ctx, tx, func(q *db.Queries) (db.Account, error) {
		partyID, err := db.CreatePersonWithParty(ctx, q)
		if err != nil {
			return db.Account{}, err
		}
		account, err := q.CreateAccount(ctx, db.CreateAccountParams{
			PersonID: partyID, Email: email,
		})
		if err != nil {
			return db.Account{}, err
		}
		if err := q.CreateAccountSettings(ctx, account.ID); err != nil {
			return db.Account{}, err
		}
		account, err = q.MarkEmailVerified(ctx, db.MarkEmailVerifiedParams{
			ID: account.ID, Email: email,
		})
		if err != nil {
			return db.Account{}, err
		}
		_, err = q.LinkOAuth2Credential(ctx, db.LinkOAuth2CredentialParams{
			Issuer: key.Issuer, Subject: key.Subject, AccountID: account.ID,
		})
		return account, err
	})
	if emailLoginTaken(err) {
		return nil, util.WithCause(ErrEmailRegistered, err)
	}
	if err != nil {
		return nil, err
	}
	return AccountFromDB(created), nil
}

// GetConnections lists recent login evidence scoped to the owning Account.
// SQLC enforces AccountID in the WHERE predicate and bounds the result set.
func GetConnections(ctx context.Context, store interface {
	ListAccountConnectionHistories(context.Context, int64) ([]db.ListAccountConnectionHistoriesRow, error)
}, accountID int64) ([]Connection, error) {
	if accountID <= 0 {
		return nil, ErrAccountUnavailable
	}
	rows, err := store.ListAccountConnectionHistories(ctx, accountID)
	if err != nil {
		return nil, err
	}
	return connectionsFromDB(rows), nil
}
