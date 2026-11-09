package org

import (
	"context"

	auth_model "github.com/datvtph41107/bdspro/models/auth"
	db "github.com/datvtph41107/bdspro/models/db"
	org_model "github.com/datvtph41107/bdspro/models/organization"
	"github.com/google/uuid"
)

// organizationActorContext exercises the same request-scoped dependencies
// installed at HTTP ingress; pinned stores preserve real lock concurrency.
func organizationActorContext(ctx context.Context, store db.Store, accountID int64, sessionID uuid.UUID) context.Context {
	if store != nil {
		ctx = db.WithStore(ctx, store)
	}
	// Middleware cannot install a malformed acting principal. Model the
	// rejected ingress state by leaving the actor absent, not by invoking
	// WithActor with a nil session (which deliberately panics).
	if accountID <= 0 || sessionID == uuid.Nil {
		return ctx
	}
	return auth_model.WithActor(ctx, auth_model.Actor{
		AccountID: accountID, SessionID: sessionID,
	})
}

// These fixture helpers adapt test data to the public DOMAIN model actions.
// They never duplicate persistence or transaction behavior.
func createOrganizationAs(ctx context.Context, name string) (*org_model.Organization, error) {
	org := &org_model.Organization{Name: name}
	if err := NewOrganization(ctx, org); err != nil {
		return nil, err
	}
	return org, nil
}

func editOrganizationAs(ctx context.Context, orgID int64, name string) (*org_model.Organization, error) {
	org := &org_model.Organization{ID: orgID, Name: name}
	if err := EditOrganization(ctx, org); err != nil {
		return nil, err
	}
	return org, nil
}
