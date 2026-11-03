// Package organization is a temporary compatibility boundary for the R01
// Gin handlers. Persistent operations now live in models/organization,
// matching Gitea's organization model ownership.
//
// Remove this adapter after all organization routes use routers/api/v1/org.
// Do not add new business or database rules here.
package organization

import (
	"context"

	db "github.com/datvtph41107/bdspro/models/db"
	org_model "github.com/datvtph41107/bdspro/models/organization"
)

type OrganizationService struct {
	store db.Store
}

func NewOrganizationService(store db.Store) *OrganizationService {
	return &OrganizationService{store: store}
}

// OrganizationAccess is kept as an alias for the existing response converters.
// No second representation or permissions model is introduced.
type OrganizationAccess = org_model.Access

func (s *OrganizationService) Create(ctx context.Context, actorAccountID int64, name string) (db.Organization, error) {
	return org_model.CreateOrganization(ctx, s.store, actorAccountID, name)
}

func (s *OrganizationService) List(ctx context.Context, actorAccountID int64) ([]OrganizationAccess, error) {
	return org_model.ListForAccount(ctx, s.store, actorAccountID)
}

func (s *OrganizationService) Get(ctx context.Context, actorAccountID, organizationID int64) (OrganizationAccess, error) {
	return org_model.GetForAccount(ctx, s.store, actorAccountID, organizationID)
}

func (s *OrganizationService) Members(ctx context.Context, actorAccountID, organizationID int64) ([]db.ListOrganizationMembersForOwnerRow, error) {
	return org_model.GetOrgMembers(ctx, s.store, actorAccountID, organizationID)
}

func (s *OrganizationService) AddMember(ctx context.Context, actorAccountID, organizationID, personID int64) (db.OrganizationMember, error) {
	return org_model.AddOrgUser(ctx, s.store, actorAccountID, organizationID, personID)
}

func (s *OrganizationService) Update(ctx context.Context, actorAccountID, organizationID int64, name string) (db.Organization, error) {
	return org_model.UpdateOrganization(ctx, s.store, actorAccountID, organizationID, name)
}
