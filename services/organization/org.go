package organization

import (
	"context"

	audit_model "github.com/datvtph41107/bdspro/models/audit"
	auth_model "github.com/datvtph41107/bdspro/models/auth"
	db "github.com/datvtph41107/bdspro/models/db"
	organization_model "github.com/datvtph41107/bdspro/models/organization"
	"github.com/google/uuid"
)

// NewOrganization creates a Party-backed Organization from an API model.
// The original service enforces account-first/session-second lock ordering;
// the caller's model is published only after a successful COMMIT.
func NewOrganization(ctx context.Context, target *organization_model.Organization) error {
	if target == nil {
		return organization_model.ErrInvalidOrganization
	}
	actor, ok := auth_model.ActorFromContext(ctx)
	if !ok {
		return auth_model.ErrSessionUnavailable
	}
	store, err := db.StoreFromContext(ctx)
	if err != nil {
		return err
	}
	created, err := CreateOrganization(ctx, store, actor.AccountID, actor.SessionID, target.Name)
	if err != nil {
		return err
	}
	*target = *organization_model.OrganizationFromDB(created)
	return nil
}

// EditOrganization retains SQLC owner-scoped writes and session authority
// through the existing UpdateOrganization transaction.
func EditOrganization(ctx context.Context, target *organization_model.Organization) error {
	if target == nil || target.ID <= 0 {
		return organization_model.ErrInvalidOrganization
	}
	actor, ok := auth_model.ActorFromContext(ctx)
	if !ok {
		return auth_model.ErrSessionUnavailable
	}
	store, err := db.StoreFromContext(ctx)
	if err != nil {
		return err
	}
	updated, err := UpdateOrganization(ctx, store, actor.AccountID, actor.SessionID, target.ID, target.Name)
	if err != nil {
		return err
	}
	*target = *organization_model.OrganizationFromDB(updated)
	return nil
}

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

// RemoveOrganizationMember coordinates the authenticated actor, owner-scoped
// ordinary member closure, team access cleanup and durable audit in one commit.
// Role changes and owner removal intentionally have a separate contract.
func RemoveOrganizationMember(ctx context.Context, store db.Store, actorAccountID int64, actingSessionID uuid.UUID, organizationID, personID int64) error {
	if organizationID <= 0 || personID <= 0 {
		return organization_model.ErrInvalidOrganization
	}
	return store.ExecTx(ctx, func(q *db.Queries) error {
		if err := auth_model.RequireActingSession(ctx, q, actorAccountID, actingSessionID); err != nil {
			return err
		}
		// Serialize ordinary removal with target promotion. Otherwise
		// remove's target row lock may invert with an owner transition.
		if err := organization_model.LockRoleChangeOrganization(ctx, q, organizationID); err != nil {
			return err
		}
		return organization_model.RemoveOrdinaryOrgMemberTx(ctx, q, actorAccountID, organizationID, personID)
	})
}

// PromoteOrganizationOwner grants ownership only to a verified current member.
// The ordinary team role and its grants are unchanged.
func PromoteOrganizationOwner(ctx context.Context, store db.Store, actorAccountID int64, sessionID uuid.UUID, organizationID, targetPersonID int64) (db.OrganizationMember, error) {
	return changeOrganizationOwner(ctx, store, actorAccountID, sessionID, organizationID, targetPersonID, ownerPromote)
}

// DemoteOrganizationOwner retains the member but removes the owner role.
// Even concurrent demotions cannot remove the final active owner.
func DemoteOrganizationOwner(ctx context.Context, store db.Store, actorAccountID int64, sessionID uuid.UUID, organizationID, targetPersonID int64) error {
	_, err := changeOrganizationOwner(ctx, store, actorAccountID, sessionID, organizationID, targetPersonID, ownerDemote)
	return err
}

// TransferOrganizationOwner makes the recipient an owner and demotes the
// caller in one commit. The old owner stays an ordinary organization member.
func TransferOrganizationOwner(ctx context.Context, store db.Store, actorAccountID int64, sessionID uuid.UUID, organizationID, targetPersonID int64) (db.OrganizationMember, error) {
	return changeOrganizationOwner(ctx, store, actorAccountID, sessionID, organizationID, targetPersonID, ownerTransfer)
}

type ownerChangeKind uint8

const (
	ownerPromote ownerChangeKind = iota
	ownerDemote
	ownerTransfer
)

func changeOrganizationOwner(ctx context.Context, store db.Store, actorAccountID int64, sessionID uuid.UUID, organizationID, targetPersonID int64, kind ownerChangeKind) (db.OrganizationMember, error) {
	if actorAccountID <= 0 || organizationID <= 0 || targetPersonID <= 0 {
		return db.OrganizationMember{}, organization_model.ErrInvalidOrganization
	}
	return db.WithTxResult(ctx, store, func(q *db.Queries) (db.OrganizationMember, error) {
		if err := auth_model.RequireActingSession(ctx, q, actorAccountID, sessionID); err != nil {
			return db.OrganizationMember{}, err
		}
		if err := organization_model.LockRoleChangeOrganization(ctx, q, organizationID); err != nil {
			return db.OrganizationMember{}, err
		}
		actorPersonID, err := organization_model.RequireRoleChangeOwner(ctx, q, organizationID, actorAccountID)
		if err != nil {
			return db.OrganizationMember{}, err
		}
		var changed db.OrganizationMember
		var action string
		switch kind {
		case ownerPromote:
			if targetPersonID == actorPersonID {
				return db.OrganizationMember{}, organization_model.ErrOwnerCandidateInvalid
			}
			changed, err = organization_model.PromoteOwnerTx(ctx, q, organizationID, targetPersonID)
			action = audit_model.ActionOrganizationOwnerPromoted
		case ownerDemote:
			var count int64
			count, err = q.CountActiveOrganizationOwners(ctx, organizationID)
			if err != nil {
				return db.OrganizationMember{}, err
			}
			if count <= 1 {
				return db.OrganizationMember{}, organization_model.ErrLastOrganizationOwner
			}
			changed, err = organization_model.DemoteOwnerTx(ctx, q, organizationID, targetPersonID)
			action = audit_model.ActionOrganizationOwnerDemoted
		case ownerTransfer:
			if targetPersonID == actorPersonID {
				return db.OrganizationMember{}, organization_model.ErrInvalidOrganization
			}
			// Critical lock order: team writes lock owner membership before
			// recipient membership. Demote actor FIRST, promote target SECOND.
			// If promotion fails, the transaction rolls the demotion back.
			if _, err := organization_model.DemoteOwnerTx(ctx, q, organizationID, actorPersonID); err != nil {
				return db.OrganizationMember{}, err
			}
			changed, err = organization_model.PromoteOwnerTx(ctx, q, organizationID, targetPersonID)
			action = audit_model.ActionOrganizationOwnerTransferred
		default:
			return db.OrganizationMember{}, organization_model.ErrInvalidOrganization
		}
		if err != nil {
			return db.OrganizationMember{}, err
		}
		if err := audit_model.RecordOrganizationOwnerRoleChanged(ctx, q, action, actorAccountID, organizationID, targetPersonID); err != nil {
			return db.OrganizationMember{}, err
		}
		return changed, nil
	})
}
