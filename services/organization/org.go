package organization

import (
	"context"

	auth_model "github.com/datvtph41107/bdspro/models/auth"
	db "github.com/datvtph41107/bdspro/models/db"
	organization_model "github.com/datvtph41107/bdspro/models/organization"
	"github.com/google/uuid"
)

// CreateOrganization composes the creator-account, acting-session and
// party/organization/owner writes in one transaction. Account FOR KEY SHARE
// MUST precede session FOR UPDATE: credential recovery takes account FOR
// UPDATE before deleting sessions. Reversing this order can deadlock.
func CreateOrganization(ctx context.Context, store db.Store, actorAccountID int64, actingSessionID uuid.UUID, name string) (db.Organization, error) {
	name, err := organization_model.OrganizationName(name)
	if err != nil {
		return db.Organization{}, err
	}
	return db.WithTxResult(ctx, store, func(q *db.Queries) (db.Organization, error) {
		creator, err := organization_model.LockCreator(ctx, q, actorAccountID)
		if err != nil {
			return db.Organization{}, err
		}
		if err := auth_model.RequireActingSession(ctx, q, actorAccountID, actingSessionID); err != nil {
			return db.Organization{}, err
		}
		return organization_model.CreateOrganizationForCreator(ctx, q, creator, name)
	})
}

// UpdateOrganization retains the SQLC owner-scoped UPDATE and its FOR SHARE
// permission lock. Session is checked and locked within the same commit.
func UpdateOrganization(ctx context.Context, store db.Store, actorAccountID int64, actingSessionID uuid.UUID, organizationID int64, name string) (db.Organization, error) {
	name, err := organization_model.OrganizationName(name)
	if err != nil {
		return db.Organization{}, err
	}
	return db.WithTxResult(ctx, store, func(q *db.Queries) (db.Organization, error) {
		if err := auth_model.RequireActingSession(ctx, q, actorAccountID, actingSessionID); err != nil {
			return db.Organization{}, err
		}
		return organization_model.UpdateOrganization(ctx, q, actorAccountID, organizationID, name)
	})
}

// AddOrganizationMember keeps owner-scoped INSERT authorization and its
// membership FOR SHARE lock in the same transaction as acting session.
// Do not preflight ownership in HTTP or rely solely on JWT claims.
func AddOrganizationMember(ctx context.Context, store db.Store, actorAccountID int64, actingSessionID uuid.UUID, organizationID, personID int64) (db.OrganizationMember, error) {
	return db.WithTxResult(ctx, store, func(q *db.Queries) (db.OrganizationMember, error) {
		if err := auth_model.RequireActingSession(ctx, q, actorAccountID, actingSessionID); err != nil {
			return db.OrganizationMember{}, err
		}
		return organization_model.AddOrgUser(ctx, q, actorAccountID, organizationID, personID)
	})
}
