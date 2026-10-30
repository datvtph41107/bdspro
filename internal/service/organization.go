package service

import (
	"context"
	"errors"
	"strings"
	"time"

	db "github.com/datvtph41107/bdspro/internal/database"
	apperrors "github.com/datvtph41107/bdspro/internal/errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// OrganizationService owns business rules and database error classification.
type OrganizationService struct {
	store db.Store
}

func NewOrganizationService(store db.Store) *OrganizationService {
	return &OrganizationService{store: store}
}

func (s *OrganizationService) personIDForAccount(ctx context.Context, accountID int64) (int64, error) {
	account, err := s.store.GetAccountByID(ctx, accountID)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, apperrors.AccountUnavailable
	}
	if err != nil {
		return 0, err
	}
	return account.PersonID, nil
}

func organizationName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", apperrors.NameRequired
	}
	return name, nil
}

func (s *OrganizationService) Create(ctx context.Context, actorAccountID int64, name string) (db.Organization, error) {
	name, err := organizationName(name)
	if err != nil {
		return db.Organization{}, err
	}
	personID, err := s.personIDForAccount(ctx, actorAccountID)
	if err != nil {
		return db.Organization{}, err
	}
	return s.store.CreateOrganizationTx(ctx, db.CreateOrganizationTxParams{
		Name: name, OwnerPersonID: personID,
	})
}

// OrganizationAccess projects membership and capabilities for a client.
// This projection cannot authorize mutations: resource SQL predicates do.
type OrganizationAccess struct {
	Organization db.Organization
	MembershipID int64
	Role         string
	JoinedAt     time.Time
	Permissions  []string
}

func organizationPermissions(role string) []string {
	switch role {
	case "owner":
		return []string{"organization.view", "organization.update", "organization.members.view", "organization.members.add"}
	case "member":
		return []string{"organization.view"}
	default:
		return []string{}
	}
}

func (s *OrganizationService) List(ctx context.Context, actorAccountID int64) ([]OrganizationAccess, error) {
	rows, err := s.store.ListOrganizationsForAccount(ctx, actorAccountID)
	if err != nil {
		return nil, err
	}
	out := make([]OrganizationAccess, 0, len(rows))
	for _, row := range rows {
		out = append(out, OrganizationAccess{
			Organization: db.Organization{PartyID: row.PartyID, Name: row.Name, CreatedAt: row.CreatedAt},
			MembershipID: row.MembershipID, Role: row.Role, JoinedAt: row.JoinedAt,
			Permissions: organizationPermissions(row.Role),
		})
	}
	return out, nil
}

func (s *OrganizationService) Get(ctx context.Context, actorAccountID, organizationID int64) (OrganizationAccess, error) {
	row, err := s.store.GetOrganizationForAccount(ctx, db.GetOrganizationForAccountParams{
		AccountID: actorAccountID, OrganizationID: organizationID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return OrganizationAccess{}, apperrors.OrganizationNotFound
	}
	if err != nil {
		return OrganizationAccess{}, err
	}
	return OrganizationAccess{
		Organization: db.Organization{PartyID: row.PartyID, Name: row.Name, CreatedAt: row.CreatedAt},
		MembershipID: row.MembershipID, Role: row.Role, JoinedAt: row.JoinedAt,
		Permissions: organizationPermissions(row.Role),
	}, nil
}

func (s *OrganizationService) Members(ctx context.Context, actorAccountID, organizationID int64) ([]db.ListOrganizationMembersForOwnerRow, error) {
	members, err := s.store.ListOrganizationMembersForOwner(ctx, db.ListOrganizationMembersForOwnerParams{
		OrganizationID: organizationID, ActorAccountID: actorAccountID,
	})
	if err != nil {
		return nil, err
	}
	// Every active owner is itself an active member. No result therefore
	// means the actor has no owner access, or the organization is absent.
	if len(members) == 0 {
		return nil, apperrors.OrganizationNotFound
	}
	return members, nil
}

func (s *OrganizationService) AddMember(ctx context.Context, actorAccountID, organizationID, personID int64) (db.OrganizationMember, error) {
	member, err := s.store.AddOrganizationMember(ctx, db.AddOrganizationMemberParams{
		OrganizationID: organizationID,
		PersonID:       personID,
		ActorAccountID: actorAccountID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return db.OrganizationMember{}, apperrors.OrganizationOwnerRequired
	}
	if pgErr := apperrors.Postgres(err); pgErr != nil {
		switch pgErr.Code {
		case "23505":
			return db.OrganizationMember{}, apperrors.Wrap(apperrors.MembershipExists, err)
		case "23503":
			return db.OrganizationMember{}, apperrors.Wrap(apperrors.PersonUnavailable, err)
		}
	}
	return member, err
}

func (s *OrganizationService) Update(ctx context.Context, actorAccountID, organizationID int64, name string) (db.Organization, error) {
	name, err := organizationName(name)
	if err != nil {
		return db.Organization{}, err
	}
	organization, err := s.store.UpdateOrganization(ctx, db.UpdateOrganizationParams{
		OrganizationID: organizationID,
		ActorAccountID: actorAccountID,
		Name:           pgtype.Text{String: name, Valid: true},
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return db.Organization{}, apperrors.OrganizationOwnerRequired
	}
	return organization, err
}
