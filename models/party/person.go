// Package party owns Party/Person persistence composition. The caller owns
// transaction lifetime through models/db.WithTx or WithTx2.
package party

import (
	"context"
	"fmt"

	"github.com/datvtph41107/bdspro/models/db"
)

// CreatePersonWithParty creates a Party root and its Person subtype using the
// caller's SQLC transaction. Account registration, OIDC-first and phone-first
// signup must all preserve this same persistence invariant.
func CreatePersonWithParty(ctx context.Context, q *db.Queries) (int64, error) {
	partyID, err := q.CreateParty(ctx)
	if err != nil {
		return 0, fmt.Errorf("create person party: %w", err)
	}
	if err := q.CreatePerson(ctx, partyID); err != nil {
		return 0, fmt.Errorf("create person subtype: %w", err)
	}
	if err := q.CreatePersonProfile(ctx, partyID); err != nil {
		return 0, fmt.Errorf("create person profile: %w", err)
	}
	return partyID, nil
}
