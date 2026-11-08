package organization

import (
	"context"
)

// OwnerCoverageReader holds only the SQLC query needed by this read-only
// diagnostic. It does not grant account or organization mutation capability.
type OwnerCoverageReader interface {
	ListOrganizationsWithoutActiveOwner(context.Context) ([]int64, error)
}

// OwnerCoverage lists at most 20 affected organizations. HasMore means there
// are additional results and the count must NOT be reported as exact.
type OwnerCoverage struct {
	OrganizationIDs []int64
	HasMore         bool
}

const maxReportedOwnerlessOrganizations = 20

// CheckOwnerCoverage diagnoses an out-of-band last-owner demotion. It does not
// try to construct a replacement owner or mutate data. Repair requires
// independently reviewed authority proof and a separate explicit operation.
func CheckOwnerCoverage(ctx context.Context, reader OwnerCoverageReader) (OwnerCoverage, error) {
	ids, err := reader.ListOrganizationsWithoutActiveOwner(ctx)
	if err != nil {
		return OwnerCoverage{}, err
	}
	if len(ids) > maxReportedOwnerlessOrganizations {
		return OwnerCoverage{
			OrganizationIDs: ids[:maxReportedOwnerlessOrganizations],
			HasMore:         true,
		}, nil
	}
	return OwnerCoverage{OrganizationIDs: ids}, nil
}
