// Package org coordinates Organization business operations across models.
// The package API follows Gitea: context-scoped actor and transaction,
// domain model arguments, and no Store/SQLC executor in public signatures.
package org

import (
	"context"

	audit_model "github.com/datvtph41107/bdspro/models/audit"
	auth_model "github.com/datvtph41107/bdspro/models/auth"
	db "github.com/datvtph41107/bdspro/models/db"
	organization_model "github.com/datvtph41107/bdspro/models/organization"
)

// actingOrganization reads the authenticated principal, not a permission
// snapshot. Live session locks and SQL authorization remain authoritative.
func actingOrganization(ctx context.Context) (auth_model.Actor, error) {
	actor, ok := auth_model.ActorFromContext(ctx)
	if !ok {
		return auth_model.Actor{}, auth_model.ErrSessionUnavailable
	}
	return actor, nil
}

// NewOrganization creates the Party, Organization and initial Owner atomically.
// The account FOR KEY SHARE lock MUST precede the session FOR UPDATE lock:
// credential recovery locks the account before deleting its sessions.
func NewOrganization(ctx context.Context, target *organization_model.Organization) error {
	if target == nil {
		return organization_model.ErrInvalidOrganization
	}
	name, err := organization_model.OrganizationName(target.Name)
	if err != nil {
		return err
	}
	actor, err := actingOrganization(ctx)
	if err != nil {
		return err
	}
	created, err := db.WithTx2(ctx, func(txCtx context.Context) (*organization_model.Organization, error) {
		q, err := db.GetQueries(txCtx)
		if err != nil {
			return nil, err
		}
		creator, err := organization_model.LockCreator(txCtx, q, actor.AccountID)
		if err != nil {
			return nil, err
		}
		if err := auth_model.RequireActingSession(txCtx, q, actor.AccountID, actor.SessionID); err != nil {
			return nil, err
		}
		row, err := organization_model.CreateOrganizationForCreator(txCtx, q, creator, name)
		if err != nil {
			return nil, err
		}
		return row, nil
	})
	if err != nil {
		return err
	}
	// A nested WithTx2 returns provisional data; only its parent can commit.
	*target = *created
	return nil
}

// EditOrganization retains SQLC owner-scoped UPDATE authorization in the same
// transaction as its acting-session row lock.
func EditOrganization(ctx context.Context, target *organization_model.Organization) error {
	if target == nil || target.ID <= 0 {
		return organization_model.ErrInvalidOrganization
	}
	name, err := organization_model.OrganizationName(target.Name)
	if err != nil {
		return err
	}
	actor, err := actingOrganization(ctx)
	if err != nil {
		return err
	}
	updated, err := db.WithTx2(ctx, func(txCtx context.Context) (*organization_model.Organization, error) {
		q, err := db.GetQueries(txCtx)
		if err != nil {
			return nil, err
		}
		if err := auth_model.RequireActingSession(txCtx, q, actor.AccountID, actor.SessionID); err != nil {
			return nil, err
		}
		row, err := organization_model.UpdateOrganization(txCtx, q, actor.AccountID, target.ID, name)
		if err != nil {
			return nil, err
		}
		return row, nil
	})
	if err != nil {
		return err
	}
	*target = *updated
	return nil
}

// AddMember uses the owner-guarded SQL INSERT and the live acting-session
// check in the same transaction. The HTTP boundary never grants membership.
func AddMember(ctx context.Context, organizationID, personID int64) (*organization_model.Member, error) {
	actor, err := actingOrganization(ctx)
	if err != nil {
		return nil, err
	}
	return db.WithTx2(ctx, func(txCtx context.Context) (*organization_model.Member, error) {
		q, err := db.GetQueries(txCtx)
		if err != nil {
			return nil, err
		}
		if err := auth_model.RequireActingSession(txCtx, q, actor.AccountID, actor.SessionID); err != nil {
			return nil, err
		}
		return organization_model.AddOrgUser(txCtx, q, actor.AccountID, organizationID, personID)
	})
}

// RemoveMember serializes ordinary removal with role changes, and commits
// team-grant cleanup plus audit with the membership closure.
func RemoveMember(ctx context.Context, organizationID, personID int64) error {
	if organizationID <= 0 || personID <= 0 {
		return organization_model.ErrInvalidOrganization
	}
	actor, err := actingOrganization(ctx)
	if err != nil {
		return err
	}
	return db.WithTx(ctx, func(txCtx context.Context) error {
		q, err := db.GetQueries(txCtx)
		if err != nil {
			return err
		}
		if err := auth_model.RequireActingSession(txCtx, q, actor.AccountID, actor.SessionID); err != nil {
			return err
		}
		// Lock organization role state before target membership to avoid
		// inversion against concurrent promotion and transfer.
		if err := organization_model.LockRoleChangeOrganization(txCtx, q, organizationID); err != nil {
			return err
		}
		return organization_model.RemoveOrdinaryOrgMemberTx(txCtx, q, actor.AccountID, organizationID, personID)
	})
}

// PromoteOwner grants ownership only to an eligible verified current member.
func PromoteOwner(ctx context.Context, organizationID, personID int64) (*organization_model.Member, error) {
	return changeOrganizationOwner(ctx, organizationID, personID, ownerPromote)
}

// DemoteOwner must not remove the final active organization owner.
func DemoteOwner(ctx context.Context, organizationID, personID int64) error {
	_, err := changeOrganizationOwner(ctx, organizationID, personID, ownerDemote)
	return err
}

// TransferOwner demotes the actor before promoting the recipient, with audit
// and all role changes committed or rolled back together.
func TransferOwner(ctx context.Context, organizationID, personID int64) (*organization_model.Member, error) {
	return changeOrganizationOwner(ctx, organizationID, personID, ownerTransfer)
}

type ownerChangeKind uint8

const (
	ownerPromote ownerChangeKind = iota
	ownerDemote
	ownerTransfer
)

func changeOrganizationOwner(ctx context.Context, organizationID, targetPersonID int64, kind ownerChangeKind) (*organization_model.Member, error) {
	if organizationID <= 0 || targetPersonID <= 0 {
		return nil, organization_model.ErrInvalidOrganization
	}
	actor, err := actingOrganization(ctx)
	if err != nil {
		return nil, err
	}
	return db.WithTx2(ctx, func(txCtx context.Context) (*organization_model.Member, error) {
		q, err := db.GetQueries(txCtx)
		if err != nil {
			return nil, err
		}
		if err := auth_model.RequireActingSession(txCtx, q, actor.AccountID, actor.SessionID); err != nil {
			return nil, err
		}
		if err := organization_model.LockRoleChangeOrganization(txCtx, q, organizationID); err != nil {
			return nil, err
		}
		actorPersonID, err := organization_model.RequireRoleChangeOwner(txCtx, q, organizationID, actor.AccountID)
		if err != nil {
			return nil, err
		}
		var changed *organization_model.Member
		var action string
		switch kind {
		case ownerPromote:
			if targetPersonID == actorPersonID {
				return nil, organization_model.ErrOwnerCandidateInvalid
			}
			changed, err = organization_model.PromoteOwnerTx(txCtx, q, organizationID, targetPersonID)
			action = audit_model.ActionOrganizationOwnerPromoted
		case ownerDemote:
			var count int64
			count, err = q.CountActiveOrganizationOwners(txCtx, organizationID)
			if err != nil {
				return nil, err
			}
			if count <= 1 {
				return nil, organization_model.ErrLastOrganizationOwner
			}
			changed, err = organization_model.DemoteOwnerTx(txCtx, q, organizationID, targetPersonID)
			action = audit_model.ActionOrganizationOwnerDemoted
		case ownerTransfer:
			if targetPersonID == actorPersonID {
				return nil, organization_model.ErrInvalidOrganization
			}
			// Team writes lock owner membership before recipient membership.
			// Demote actor FIRST, promote target SECOND. A failed promotion
			// must roll the first step back.
			if _, err := organization_model.DemoteOwnerTx(txCtx, q, organizationID, actorPersonID); err != nil {
				return nil, err
			}
			changed, err = organization_model.PromoteOwnerTx(txCtx, q, organizationID, targetPersonID)
			action = audit_model.ActionOrganizationOwnerTransferred
		default:
			return nil, organization_model.ErrInvalidOrganization
		}
		if err != nil {
			return nil, err
		}
		if err := audit_model.RecordOrganizationOwnerRoleChanged(txCtx, q, action, actor.AccountID, organizationID, targetPersonID); err != nil {
			return nil, err
		}
		return changed, nil
	})
}
