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
func AddTeamUser(ctx context.Context, writer interface {
	AddTeamUserForOwner(context.Context, db.AddTeamUserForOwnerParams) (db.TeamMember, error)
}, actorAccountID, teamID, personID int64) (db.TeamMember, error) {
	member, err := writer.AddTeamUserForOwner(ctx, db.AddTeamUserForOwnerParams{
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
	members, err := store.ListTeamUsersForAccount(ctx, db.ListTeamUsersForAccountParams{
		ActorAccountID: actorAccountID, TeamID: teamID,
		Take: take, Skip: skip,
	})
	if err != nil || len(members) != 0 {
		return members, err
	}
	// A zero-row SELECT cannot distinguish an empty team from an invisible
	// team. Check resource visibility only in the empty-result case.
	if _, err := GetTeam(ctx, store, actorAccountID, teamID); err != nil {
		return nil, err
	}
	return members, nil
}

// RemoveTeamUser retains the standalone transaction contract for model callers.
// API writes use the service-owned transaction to include session authority.
func RemoveTeamUser(ctx context.Context, store db.Store, actorAccountID, teamID, personID int64) error {
	return store.ExecTx(ctx, func(q *db.Queries) error {
		return RemoveTeamUserTx(ctx, q, actorAccountID, teamID, personID)
	})
}

// RemoveTeamUserTx owns owner-scoped SQLC deletion and its audit evidence.
// The caller must execute it inside a transaction; both operations roll back
// together if either fails.
func RemoveTeamUserTx(ctx context.Context, q *db.Queries, actorAccountID, teamID, personID int64) error {
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
}
