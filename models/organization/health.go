package organization

import (
	"context"
	"fmt"

	db "github.com/datvtph41107/bdspro/models/db"
)

// OwnerCoverage lists at most 20 affected organizations. HasMore means there
// are additional results and the count must NOT be reported as exact.
type OwnerCoverage struct {
	OrganizationIDs []int64
	HasMore         bool
}

const maxReportedOwnerlessOrganizations = 20

// CheckOwnerCoverage diagnoses an out-of-band last-owner demotion.
// afterID is an exclusive lower bound, allowing bounded operator inspection.
// It does not try to construct a replacement owner or mutate data. Repair requires
// independently reviewed authority proof and a separate explicit operation.
func CheckOwnerCoverage(ctx context.Context, afterID int64) (OwnerCoverage, error) {
	if afterID < 0 {
		return OwnerCoverage{}, fmt.Errorf("owner coverage after-id must not be negative")
	}
	q, err := db.GetQueries(ctx)
	if err != nil {
		return OwnerCoverage{}, err
	}
	ids, err := q.ListOrganizationsWithoutActiveOwner(ctx, afterID)
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
