package organization

import (
	"context"
	"errors"
	"math"

	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// AddTeamUser requires an active Organization owner and an active Organization member.
func AddTeamUser(ctx context.Context, store db.Store, actorAccountID, teamID, personID int64) (db.TeamMember, error) {
	member, err := store.AddTeamUserForOwner(ctx, db.AddTeamUserForOwnerParams{
		TeamID: teamID, ActorAccountID: actorAccountID, PersonID: personID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return db.TeamMember{}, ErrTeamOwnerRequired
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "team_members_pkey" {
		return db.TeamMember{}, errors.Join(ErrTeamMemberExists, err)
	}
	return member, err
}

// ListTeamUsers enforces bounded SQL pagination and active organization membership.
func ListTeamUsers(ctx context.Context, store db.Store, actorAccountID, teamID int64, opts db.ListOptions) ([]db.TeamMember, error) {
	if opts.IsListAll() {
		return nil, ErrTeamPaginationInvalid
	}
	skip, take := opts.GetSkipTake()
	if take <= 0 || take > math.MaxInt32 || skip < 0 || skip > math.MaxInt32 {
		return nil, ErrTeamPaginationInvalid
	}
	return store.ListTeamUsersForAccount(ctx, db.ListTeamUsersForAccountParams{
		ActorAccountID: actorAccountID, TeamID: teamID,
		Take: int32(take), Skip: int32(skip),
	})
}

func RemoveTeamUser(ctx context.Context, store db.Store, actorAccountID, teamID, personID int64) error {
	count, err := store.RemoveTeamUserForOwner(ctx, db.RemoveTeamUserForOwnerParams{
		TeamID: teamID, ActorAccountID: actorAccountID, PersonID: personID,
	})
	if err != nil {
		return err
	}
	if count != 1 {
		return ErrTeamMemberNotFound
	}
	return nil
}
