package db

import "context"

type CreateOrganizationTxParams struct {
	Name          string
	OwnerPersonID int64
}

func (s *SQLStore) CreateOrganizationTx(ctx context.Context, arg CreateOrganizationTxParams) (Organization, error) {
	var organization Organization
	err := s.execTx(ctx, func(q *Queries) error {
		partyID, err := q.CreateParty(ctx)
		if err != nil { return err }
		organization, err = q.CreateOrganization(ctx, CreateOrganizationParams{
			PartyID: partyID, Name: arg.Name,
		})
		if err != nil { return err }
		_, err = q.CreateOrganizationMembership(ctx, CreateOrganizationMembershipParams{
			OrganizationID: partyID, PersonID: arg.OwnerPersonID, Role: "owner",
		})
		return err
	})
	return organization, err
}
