package services

import (
	"context"

	db "github.com/datvtph41107/bdspro/db/sqlc"
	"github.com/jackc/pgx/v5/pgtype"
)

// OrganizationService is the single business entrypoint for Organization HTTP operations.
// It owns operation sequencing; SQLC enforces atomic owner predicates and transactions.
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

func (s *OrganizationService) Create(ctx context.Context, actorAccountID int64, name string) (db.Organization, error) {
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
	return s.store.AddOrganizationMember(ctx, db.AddOrganizationMemberParams{
		OrganizationID: organizationID,
		PersonID: personID,
		ActorAccountID: actorAccountID,
	})
}

func (s *OrganizationService) Update(ctx context.Context, actorAccountID, organizationID int64, name string) (db.Organization, error) {
	return s.store.UpdateOrganization(ctx, db.UpdateOrganizationParams{
		OrganizationID: organizationID,
		ActorAccountID: actorAccountID,
		Name: pgtype.Text{String: name, Valid: true},
	})
}
