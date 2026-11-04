package organization

import (
	"context"

	db "github.com/datvtph41107/bdspro/models/db"
)

// ListTeams applies the same authorization predicate as the single-team query.
func ListTeams(ctx context.Context, store db.Store, actorAccountID, organizationID int64, opts db.ListOptions) ([]db.Team, error) {
	skip, take, ok := opts.GetSkipTake32()
	if !ok {
		return nil, ErrTeamPaginationInvalid
	}
	return store.ListTeamsForAccount(ctx, db.ListTeamsForAccountParams{
		ActorAccountID: actorAccountID,
		OrganizationID: organizationID,
		Take:           take,
		Skip:           skip,
	})
}
