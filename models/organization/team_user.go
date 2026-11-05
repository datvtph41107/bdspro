package organization

import (
	"context"
	"errors"

	audit_model "github.com/datvtph41107/bdspro/models/audit"
	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/modules/util"
	"github.com/jackc/pgx/v5"
)

// AddTeamUser requires an active Organization owner and an active Organization member.
func AddTeamUser(ctx context.Context, store db.Store, actorAccountID, teamID, personID int64) (db.TeamMember, error) {
	member, err := store.AddTeamUserForOwner(ctx, db.AddTeamUserForOwnerParams{
		TeamID: teamID, ActorAccountID: actorAccountID, PersonID: personID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return db.TeamMember{}, util.WithCause(ErrTeamOwnerRequired, err)
	}
	if isPGConstraint(err, "23505", "team_members_pkey") {
		return db.TeamMember{}, util.WithCause(ErrTeamMemberExists, err)
	}
	return member, err
}

// ListTeamUsers enforces bounded SQL pagination and active organization membership.
func ListTeamUsers(ctx context.Context, store db.Store, actorAccountID, teamID int64, opts db.ListOptions) ([]db.TeamMember, error) {
	skip, take, ok := opts.GetSkipTake32()
	if !ok {
		return nil, ErrTeamPaginationInvalid
	}
	return store.ListTeamUsersForAccount(ctx, db.ListTeamUsersForAccountParams{
		ActorAccountID: actorAccountID, TeamID: teamID,
		Take: take, Skip: skip,
	})
}

func RemoveTeamUser(ctx context.Context, store db.Store, actorAccountID, teamID, personID int64) error {
	// Preserve the existing owner-scoped DELETE and its concurrency locks.
	// Persist audit evidence before committing the membership removal.
	return store.ExecTx(ctx, func(q *db.Queries) error {
		count, err := q.RemoveTeamUserForOwner(ctx, db.RemoveTeamUserForOwnerParams{
			TeamID: teamID, ActorAccountID: actorAccountID, PersonID: personID,
		})
		if err != nil {
			return err
		}
		if count != 1 {
			return ErrTeamMemberNotFound
		}
		return audit_model.RecordTeamMemberRemoved(ctx, q, actorAccountID, teamID, personID)
	})
}
