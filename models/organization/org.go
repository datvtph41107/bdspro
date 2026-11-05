package organization

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	account_model "github.com/datvtph41107/bdspro/models/account"
	db "github.com/datvtph41107/bdspro/models/db"
)

// Access includes the account-scoped membership returned by SQL. It is not an
// HTTP representation and does not grant permission by itself.
type Access struct {
	Organization db.Organization
	MembershipID int64
	Role         string
	JoinedAt     time.Time
}

func organizationName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", ErrNameRequired
	}
	return name, nil
}

// CreateOrganization creates the party, organization, and initial owner in
// one PostgreSQL transaction. This is the equivalent of Gitea's
// models/organization.CreateOrganization, but using SQLC instead of XORM.
func CreateOrganization(ctx context.Context, store db.Store, actorAccountID int64, name string) (db.Organization, error) {
	name, err := organizationName(name)
	if err != nil {
		return db.Organization{}, err
	}
	created, err := db.WithTxResult(ctx, store, func(q *db.Queries) (db.Organization, error) {
		// Lock the originating account until this owner membership commits.
		// A surviving Person row must not authorize a deleted Account.
		actorPersonID, err := q.LockOrganizationCreator(ctx, actorAccountID)
		if errors.Is(err, pgx.ErrNoRows) {
			return db.Organization{}, account_model.ErrAccountUnavailable
		}
		if err != nil {
			return db.Organization{}, err
		}
		partyID, err := q.CreateParty(ctx)
		if err != nil {
			return db.Organization{}, fmt.Errorf("create party: %w", err)
		}
		organization, err := q.CreateOrganization(ctx, db.CreateOrganizationParams{
			PartyID: partyID, Name: name,
		})
		if err != nil {
			return db.Organization{}, fmt.Errorf("create organization: %w", err)
		}
		_, err = q.CreateOrganizationMember(ctx, db.CreateOrganizationMemberParams{
			OrganizationID: partyID, PersonID: actorPersonID, Role: RoleOwner,
		})
		if err != nil {
			return db.Organization{}, fmt.Errorf("create owner membership: %w", err)
		}
		return organization, nil
	})
	if err != nil {
		return db.Organization{}, fmt.Errorf("create organization: %w", err)
	}
	return created, nil
}

func ListForAccount(ctx context.Context, store db.Store, actorAccountID int64) ([]Access, error) {
	rows, err := store.ListOrganizationsForAccount(ctx, actorAccountID)
	if err != nil {
		return nil, err
	}
	out := make([]Access, 0, len(rows))
	for _, row := range rows {
		out = append(out, Access{
			Organization: db.Organization{PartyID: row.PartyID, Name: row.Name, CreatedAt: row.CreatedAt},
			MembershipID: row.MembershipID, Role: row.Role, JoinedAt: row.JoinedAt,
		})
	}
	return out, nil
}

// GetForAccount preserves the account visibility predicate in SQLC.
func GetForAccount(ctx context.Context, store db.Store, actorAccountID, organizationID int64) (Access, error) {
	row, err := store.GetOrganizationForAccount(ctx, db.GetOrganizationForAccountParams{
		AccountID: actorAccountID, OrganizationID: organizationID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Access{}, ErrOrganizationNotFound
	}
	if err != nil {
		return Access{}, err
	}
	return Access{
		Organization: db.Organization{PartyID: row.PartyID, Name: row.Name, CreatedAt: row.CreatedAt},
		MembershipID: row.MembershipID, Role: row.Role, JoinedAt: row.JoinedAt,
	}, nil
}

// UpdateOrganization must keep the owner-guarded atomic SQLC UPDATE; replacing
// this with a separate "check permission, then update" introduces a race.
func UpdateOrganization(ctx context.Context, store db.Store, actorAccountID, organizationID int64, name string) (db.Organization, error) {
	name, err := organizationName(name)
	if err != nil {
		return db.Organization{}, err
	}
	updated, err := store.UpdateOrganization(ctx, db.UpdateOrganizationParams{
		OrganizationID: organizationID, ActorAccountID: actorAccountID,
		Name: pgtype.Text{String: name, Valid: true},
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return db.Organization{}, ErrOrganizationOwnerRequired
	}
	return updated, err
}

// Stored role names are constrained by the organization_members table.
// They identify persisted membership roles, not HTTP permission metadata.
const (
	RoleOwner  = "owner"
	RoleMember = "member"
)
