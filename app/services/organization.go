package services

import (
	"context"

	db "github.com/datvtph41107/bdspro/db/sqlc"
)

// personIDForAccount resolves the actor used by Organization operations.
// Authentication has already established the AccountID; it does not establish
// Organization membership or legal ownership of any Property.
func personIDForAccount(ctx context.Context, store db.Store, accountID int64) (int64, error) {
	account, err := store.GetAccountByID(ctx, accountID)
	if err != nil {
		return 0, err
	}
	return account.PersonID, nil
}

// CreateForAccount creates an Organization and its initial owner membership.
// CreateOrganizationTx owns the PostgreSQL transaction; this operation only
// maps the authenticated account to its Person.
func CreateForAccount(ctx context.Context, store db.Store, accountID int64, name string) (db.Organization, error) {
	personID, err := personIDForAccount(ctx, store, accountID)
	if err != nil {
		return db.Organization{}, err
	}
	return store.CreateOrganizationTx(ctx, db.CreateOrganizationTxParams{
		Name: name, OwnerPersonID: personID,
	})
}

// ListForAccount returns only Organizations where the actor's Person has an
// active membership. The SQL query owns the active-membership predicate.
func ListForAccount(ctx context.Context, store db.Store, accountID int64) ([]db.Organization, error) {
	personID, err := personIDForAccount(ctx, store, accountID)
	if err != nil {
		return nil, err
	}
	return store.ListOrganizationsForPerson(ctx, personID)
}
