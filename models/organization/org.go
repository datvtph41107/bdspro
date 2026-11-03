// Package organization owns persistent organization behavior, not HTTP delivery.
// SQLC remains the sole query generator and PostgreSQL remains the source of truth.
package organization

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	db "github.com/datvtph41107/bdspro/models/db"
	apperrors "github.com/datvtph41107/bdspro/modules/errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// Store is the database capability this model actually consumes. The current
// db.Store satisfies it without a new repository implementation.
type Store interface {
	GetAccountByID(context.Context, int64) (db.Account, error)
	ExecTx(context.Context, func(*db.Queries) error) error
	ListOrganizationsForAccount(context.Context, int64) ([]db.ListOrganizationsForAccountRow, error)
	GetOrganizationForAccount(context.Context, db.GetOrganizationForAccountParams) (db.GetOrganizationForAccountRow, error)
	UpdateOrganization(context.Context, db.UpdateOrganizationParams) (db.Organization, error)
	ListOrganizationMembersForOwner(context.Context, db.ListOrganizationMembersForOwnerParams) ([]db.ListOrganizationMembersForOwnerRow, error)
	AddOrganizationMember(context.Context, db.AddOrganizationMemberParams) (db.OrganizationMember, error)
}

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
		return "", apperrors.NameRequired
	}
	return name, nil
}

// CreateOrganization creates the party, organization, and initial owner in
// one PostgreSQL transaction. This is the equivalent of Gitea's
// models/organization.CreateOrganization, but using SQLC instead of XORM.
func CreateOrganization(ctx context.Context, store Store, actorAccountID int64, name string) (db.Organization, error) {
	name, err := organizationName(name)
	if err != nil {
		return db.Organization{}, err
	}
	actor, err := store.GetAccountByID(ctx, actorAccountID)
	if errors.Is(err, pgx.ErrNoRows) {
		return db.Organization{}, apperrors.AccountUnavailable
	}
	if err != nil {
		return db.Organization{}, err
	}
	var created db.Organization
	err = store.ExecTx(ctx, func(q *db.Queries) error {
		partyID, err := q.CreateParty(ctx)
		if err != nil {
			return fmt.Errorf("create party: %w", err)
		}
		created, err = q.CreateOrganization(ctx, db.CreateOrganizationParams{PartyID: partyID, Name: name})
		if err != nil {
			return fmt.Errorf("create organization: %w", err)
		}
		_, err = q.CreateOrganizationMember(ctx, db.CreateOrganizationMemberParams{
			OrganizationID: partyID, PersonID: actor.PersonID, Role: "owner",
		})
		if err != nil {
			return fmt.Errorf("create owner membership: %w", err)
		}
		return nil
	})
	if err != nil {
		return db.Organization{}, fmt.Errorf("create organization: %w", err)
	}
	return created, nil
}

func ListForAccount(ctx context.Context, store Store, actorAccountID int64) ([]Access, error) {
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
func GetForAccount(ctx context.Context, store Store, actorAccountID, organizationID int64) (Access, error) {
	row, err := store.GetOrganizationForAccount(ctx, db.GetOrganizationForAccountParams{
		AccountID: actorAccountID, OrganizationID: organizationID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Access{}, apperrors.OrganizationNotFound
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
func UpdateOrganization(ctx context.Context, store Store, actorAccountID, organizationID int64, name string) (db.Organization, error) {
	name, err := organizationName(name)
	if err != nil {
		return db.Organization{}, err
	}
	updated, err := store.UpdateOrganization(ctx, db.UpdateOrganizationParams{
		OrganizationID: organizationID, ActorAccountID: actorAccountID,
		Name: pgtype.Text{String: name, Valid: true},
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return db.Organization{}, apperrors.OrganizationOwnerRequired
	}
	return updated, err
}

