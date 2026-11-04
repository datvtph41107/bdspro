package organization

import (
	"context"
	"math"

	db "github.com/datvtph41107/bdspro/models/db"
)

// ListTeams applies the same authorization predicate as the single-team query.
func ListTeams(ctx context.Context, store db.Store, actorAccountID, organizationID int64, opts db.ListOptions) ([]db.Team, error) {
	if opts.IsListAll() {
		return nil, ErrTeamPaginationInvalid
	}
	skip, take := opts.GetSkipTake()
	if take <= 0 || take > math.MaxInt32 || skip < 0 || skip > math.MaxInt32 {
		return nil, ErrTeamPaginationInvalid
	}
	return store.ListTeamsForAccount(ctx, db.ListTeamsForAccountParams{
		ActorAccountID: actorAccountID,
		OrganizationID: organizationID,
		Take:           int32(take),
		Skip:           int32(skip),
	})
}
