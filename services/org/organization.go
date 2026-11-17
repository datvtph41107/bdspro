package org

import (
	"context"
	"errors"

	"github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/models/organization"
	"github.com/jackc/pgx/v5"
)

// GetOrganizationForAccount only resolves a resource for a member with an
// active membership. It does not know about HTTP routers or APIContext.
func GetOrganizationForAccount(ctx context.Context, accountID, orgID int64) (*organization.Organization, error) {
	q, err := db.GetQueries(ctx)
	if err != nil {
		return nil, err
	}
	org, err := q.GetOrganizationForAccount(ctx, db.GetOrganizationForAccountParams{
		AccountID: accountID, OrganizationID: orgID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, organization.ErrOrganizationNotFound
	}
	if err != nil {
		return nil, err
	}
	return &organization.Organization{ID: org.PartyID, Name: org.Name, Role: org.Role}, nil
}
