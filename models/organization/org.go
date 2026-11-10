package organization

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	account_model "github.com/datvtph41107/bdspro/models/account"
	auth_model "github.com/datvtph41107/bdspro/models/auth"
	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// Organization is the Party-backed model visible to API actions.
// ID maps to organizations.party_id; SQLC keeps database field names.
type Organization struct {
	ID        int64
	Name      string
	CreatedAt time.Time
}

// OrganizationFromDB converts a committed SQLC record at the model boundary.
func OrganizationFromDB(row db.Organization) *Organization {
	return &Organization{
		ID:        row.PartyID,
		Name:      row.Name,
		CreatedAt: row.CreatedAt,
	}
}

// Access includes the account-scoped membership returned by SQL. It is not an
// HTTP representation and does not grant permission by itself.
type Access struct {
	Organization Organization
	MembershipID int64
	Role         string
	JoinedAt     time.Time
}

// OrganizationModel is a read projection for an already authorized API request.
func (a Access) OrganizationModel() *Organization {
	return &a.Organization
}

// ListForDoer is the account-scoped model read for a verified request.
// Worker/CLI callers explicitly install the acting principal in their context.
func ListForDoer(ctx context.Context) ([]Access, error) {
	actor, ok := auth_model.ActorFromContext(ctx)
	if !ok {
		return nil, auth_model.ErrSessionUnavailable
	}
	store, err := db.StoreFromContext(ctx)
	if err != nil {
		return nil, err
	}
	return ListForAccount(ctx, store, actor.AccountID)
}

// OrganizationName normalizes the canonical organization name. Both the
// standalone model entry point and HTTP service use this DB-free validator.
func OrganizationName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", ErrNameRequired
	}
	return name, nil
}

// LockedCreator is an opaque account-to-person authorization obtained under
// the account's PostgreSQL FOR KEY SHARE lock. Its unexported value prevents
// a caller from inventing a positive PersonID and bypassing the account check.
// It is valid ONLY with the exact transaction executor that acquired the lock.
// A copied proof cannot authorize an independently committed transaction.
type LockedCreator struct {
	personID int64
	executor *db.Queries
}

// LockCreator must run before RequireActingSession when creating organizations.
// Password reset locks the account FOR UPDATE before deleting sessions; locking
// the session first here would invert that order and permit deadlocks.
// FOR KEY SHARE also prevents account deletion during owner creation.
func LockCreator(ctx context.Context, q *db.Queries, actorAccountID int64) (LockedCreator, error) {
	if actorAccountID <= 0 {
		return LockedCreator{}, account_model.ErrAccountUnavailable
	}
	personID, err := q.LockOrganizationCreator(ctx, actorAccountID)
	if errors.Is(err, pgx.ErrNoRows) {
		return LockedCreator{}, account_model.ErrAccountUnavailable
	}
	if err != nil {
		return LockedCreator{}, err
	}
	return LockedCreator{personID: personID, executor: q}, nil
}

// CreateOrganizationForCreator completes the party/organization/owner write
// in the transaction that already holds the account lock. The session-aware
// service holds its session lock as well. There is no nested transaction.
func CreateOrganizationForCreator(ctx context.Context, q *db.Queries, creator LockedCreator, name string) (*Organization, error) {
	name, err := OrganizationName(name)
	if err != nil {
		return nil, err
	}
	if creator.personID <= 0 || creator.executor == nil || creator.executor != q {
		return nil, account_model.ErrAccountUnavailable
	}
	partyID, err := q.CreateParty(ctx)
	if err != nil {
		return nil, fmt.Errorf("create party: %w", err)
	}
	organization, err := q.CreateOrganization(ctx, db.CreateOrganizationParams{
		PartyID: partyID, Name: name,
	})
	if err != nil {
		return nil, fmt.Errorf("create organization: %w", err)
	}
	if _, err := q.CreateOrganizationMember(ctx, db.CreateOrganizationMemberParams{
		OrganizationID: partyID, PersonID: creator.personID, Role: RoleOwner,
	}); err != nil {
		return nil, fmt.Errorf("create owner membership: %w", err)
	}
	return OrganizationFromDB(organization), nil
}

// CreateOrganization preserves the trusted standalone model API. Production
// first-party HTTP calls the service, which additionally checks/locks session
// authority within this same account-first transaction.
func CreateOrganization(ctx context.Context, store db.Store, actorAccountID int64, name string) (*Organization, error) {
	name, err := OrganizationName(name)
	if err != nil {
		return nil, err
	}
	created, err := db.WithTxResult(ctx, store, func(q *db.Queries) (*Organization, error) {
		creator, err := LockCreator(ctx, q, actorAccountID)
		if err != nil {
			return nil, err
		}
		return CreateOrganizationForCreator(ctx, q, creator, name)
	})
	if err != nil {
		return nil, fmt.Errorf("create organization: %w", err)
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
			Organization: Organization{ID: row.PartyID, Name: row.Name, CreatedAt: row.CreatedAt},
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
		Organization: Organization{ID: row.PartyID, Name: row.Name, CreatedAt: row.CreatedAt},
		MembershipID: row.MembershipID, Role: row.Role, JoinedAt: row.JoinedAt,
	}, nil
}

// OrganizationWriter is implemented by both SQLStore and transaction-bound
// *db.Queries. There is no need for an outer store when the service already
// owns the transaction.
type OrganizationWriter interface {
	UpdateOrganization(context.Context, db.UpdateOrganizationParams) (db.Organization, error)
	AddOrganizationMember(context.Context, db.AddOrganizationMemberParams) (db.OrganizationMember, error)
}

// UpdateOrganization must keep the owner-guarded atomic SQLC UPDATE; replacing
// this with a separate "check permission, then update" introduces a race.
func UpdateOrganization(ctx context.Context, writer OrganizationWriter, actorAccountID, organizationID int64, name string) (*Organization, error) {
	name, err := OrganizationName(name)
	if err != nil {
		return nil, err
	}
	updated, err := writer.UpdateOrganization(ctx, db.UpdateOrganizationParams{
		OrganizationID: organizationID, ActorAccountID: actorAccountID,
		Name: pgtype.Text{String: name, Valid: true},
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrOrganizationOwnerRequired
	}
	if err != nil {
		return nil, err
	}
	return OrganizationFromDB(updated), nil
}

// Stored role names are constrained by the organization_members table.
// They identify persisted membership roles, not HTTP permission metadata.
const (
	RoleOwner  = "owner"
	RoleMember = "member"
)
