package audit

import (
	"context"
	"fmt"
	"strconv"

	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/google/uuid"
)

// Each action is a concrete, reviewable security event, not a generic bus.
// The database constraint provides the last line of validation.
const (
	ActionSessionRevoked            = "auth.session.revoked"
	ActionTeamMemberRemoved         = "team.member.removed"
	ActionOrganizationMemberRemoved = "organization.member.removed"
	ActionOrganizationOwnerPromoted = "organization.owner.promoted"
	ActionOrganizationOwnerDemoted = "organization.owner.demoted"
	ActionOrganizationOwnerTransferred = "organization.owner.transferred"
)

type actionWriter interface {
	InsertAuditAction(context.Context, db.InsertAuditActionParams) error
}

// RecordSessionRevoked is called inside the same SQL transaction that blocks
// the refresh session. The actor and scope are the owning account.
func RecordSessionRevoked(ctx context.Context, q actionWriter, accountID int64, sessionID uuid.UUID) error {
	if accountID <= 0 || sessionID == uuid.Nil {
		return fmt.Errorf("invalid session revocation audit identity")
	}
	return q.InsertAuditAction(ctx, db.InsertAuditActionParams{
		Action:         ActionSessionRevoked,
		ActorAccountID: accountID,
		ScopeType:      "account",
		ScopeRef:       strconv.FormatInt(accountID, 10),
		TargetType:     "session",
		TargetRef:      sessionID.String(),
	})
}

// RecordTeamMemberRemoved captures the authorizing account, affected team,
// and removed person, without storing email, OTP, bearer token or request JSON.
func RecordTeamMemberRemoved(ctx context.Context, q actionWriter, actorID, teamID, personID int64) error {
	if actorID <= 0 || teamID <= 0 || personID <= 0 {
		return fmt.Errorf("invalid team membership audit identity")
	}
	return q.InsertAuditAction(ctx, db.InsertAuditActionParams{
		Action:         ActionTeamMemberRemoved,
		ActorAccountID: actorID,
		ScopeType:      "team",
		ScopeRef:       strconv.FormatInt(teamID, 10),
		TargetType:     "person",
		TargetRef:      strconv.FormatInt(personID, 10),
	})
}

// RecordOrganizationMemberRemoved keeps revocation evidence atomic with its
// membership and team-link mutations. No PII or bearer secret is logged.
func RecordOrganizationMemberRemoved(ctx context.Context, q actionWriter, actorAccountID, organizationID, personID int64) error {
	if actorAccountID <= 0 || organizationID <= 0 || personID <= 0 {
		return fmt.Errorf("invalid organization removal audit identity")
	}
	return q.InsertAuditAction(ctx, db.InsertAuditActionParams{
		Action:         ActionOrganizationMemberRemoved,
		ActorAccountID: actorAccountID,
		ScopeType:      "organization",
		ScopeRef:       strconv.FormatInt(organizationID, 10),
		TargetType:     "person",
		TargetRef:      strconv.FormatInt(personID, 10),
	})
}

// RecordOrganizationOwnerRoleChanged records only a concrete and reviewed
// owner lifecycle action. Transferred means actor relinquished their owner
// role in the same transaction in which the target gained it.
func RecordOrganizationOwnerRoleChanged(ctx context.Context, q actionWriter, action string, actorAccountID, organizationID, targetPersonID int64) error {
	if actorAccountID <= 0 || organizationID <= 0 || targetPersonID <= 0 {
		return fmt.Errorf("invalid organization owner audit identity")
	}
	switch action {
	case ActionOrganizationOwnerPromoted, ActionOrganizationOwnerDemoted, ActionOrganizationOwnerTransferred:
	default:
		return fmt.Errorf("unsupported organization owner audit action")
	}
	return q.InsertAuditAction(ctx, db.InsertAuditActionParams{
		Action: action,
		ActorAccountID: actorAccountID,
		ScopeType: "organization",
		ScopeRef: strconv.FormatInt(organizationID, 10),
		TargetType: "person",
		TargetRef: strconv.FormatInt(targetPersonID, 10),
	})
}
