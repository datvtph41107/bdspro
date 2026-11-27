package party

import (
	"context"
	"fmt"

	"github.com/datvtph41107/bdspro/models/db/sqlc"
)

func CreatePersonWithParty(ctx context.Context, q *sqlc.Queries) (int64, error) {
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
