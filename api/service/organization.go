package service

import (
	"context"
	"errors"
	"strings"

	apperrors "github.com/datvtph41107/bdspro/core/errors"
	db "github.com/datvtph41107/bdspro/db/sqlc"
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

func (s *OrganizationService) List(ctx context.Context, actorAccountID int64) ([]db.Organization, error) {
	personID, err := s.personIDForAccount(ctx, actorAccountID)
	if err != nil {
		return nil, err
	}
	return s.store.ListOrganizationsForPerson(ctx, personID)
}

func (s *OrganizationService) AddMember(ctx context.Context, actorAccountID, organizationID, personID int64) (db.OrganizationMembership, error) {
	member, err := s.store.AddOrganizationMember(ctx, db.AddOrganizationMemberParams{
		OrganizationID: organizationID,
		PersonID: personID,
		ActorAccountID: actorAccountID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return db.OrganizationMembership{}, apperrors.OrganizationOwnerRequired
	}
	if pgErr := apperrors.Postgres(err); pgErr != nil {
		switch pgErr.Code {
		case "23505":
			return db.OrganizationMembership{}, apperrors.Wrap(apperrors.MembershipExists, err)
		case "23503":
			return db.OrganizationMembership{}, apperrors.Wrap(apperrors.PersonUnavailable, err)
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
		Name: pgtype.Text{String: name, Valid: true},
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return db.Organization{}, apperrors.OrganizationOwnerRequired
	}
	return organization, err
}
