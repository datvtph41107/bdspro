package organization

import (
	"context"
	"errors"
	"time"

	audit_model "github.com/datvtph41107/bdspro/models/audit"
	auth_model "github.com/datvtph41107/bdspro/models/auth"
	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/modules/util"
	"github.com/jackc/pgx/v5"
)

// TeamMember is the domain relation between a Team and a Person.
// Gitea owns its TeamUser relation in models/organization/team_user.go;
// BDSPro preserves its existing Person (rather than Account) membership key.
type TeamMember struct {
	TeamID   int64
	PersonID int64
	JoinedAt time.Time
}

// TeamMemberFromDB is the sole SQLC membership-to-model conversion.
func TeamMemberFromDB(row db.TeamMember) *TeamMember {
	return &TeamMember{
		TeamID: row.TeamID, PersonID: row.PersonID, JoinedAt: row.JoinedAt,
	}
}

// teamMembersFromDB preserves nil-vs-empty response behavior at the API boundary.
func teamMembersFromDB(rows []db.TeamMember) []*TeamMember {
	if rows == nil {
		return nil
	}
	members := make([]*TeamMember, 0, len(rows))
	for _, row := range rows {
		members = append(members, TeamMemberFromDB(row))
	}
	return members
}

// TeamMemberWriter is the one SQLC mutation needed by the team model.
// A regular store and transaction-bound Queries both satisfy it.
type TeamMemberWriter interface {
	AddTeamUserForOwner(context.Context, db.AddTeamUserForOwnerParams) (db.TeamMember, error)
}

// AddTeamUser requires an active Organization owner and an active Organization member.
func AddTeamUser(ctx context.Context, writer TeamMemberWriter, actorAccountID, teamID, personID int64) (*TeamMember, error) {
	member, err := writer.AddTeamUserForOwner(ctx, db.AddTeamUserForOwnerParams{
		TeamID: teamID, ActorAccountID: actorAccountID, PersonID: personID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, util.WithCause(ErrTeamOwnerRequired, err)
	}
	if isPGConstraint(err, "23505", "team_members_pkey") {
		return nil, util.WithCause(ErrTeamMemberExists, err)
	}
	if err != nil {
		return nil, err
	}
	return TeamMemberFromDB(member), nil
}

// ListTeamUsersForDoer resolves authority from request context, not client input.
func ListTeamUsersForDoer(ctx context.Context, teamID int64, opts db.ListOptions) ([]*TeamMember, error) {
	actor, ok := auth_model.ActorFromContext(ctx)
	if !ok {
		return nil, auth_model.ErrSessionUnavailable
	}
	store, err := db.StoreFromContext(ctx)
	if err != nil {
		return nil, err
	}
	return ListTeamUsers(ctx, store, actor.AccountID, teamID, opts)
}

// ListTeamUsers enforces bounded SQL pagination and active organization membership.
func ListTeamUsers(ctx context.Context, store db.Store, actorAccountID, teamID int64, opts db.ListOptions) ([]*TeamMember, error) {
	skip, take, ok := opts.GetSkipTake32()
	if !ok {
		return nil, ErrTeamPaginationInvalid
	}
	members, err := store.ListTeamUsersForAccount(ctx, db.ListTeamUsersForAccountParams{
		ActorAccountID: actorAccountID, TeamID: teamID,
		Take: take, Skip: skip,
	})
	if err != nil {
		return nil, err
	}
	// A zero-row SELECT cannot distinguish an empty team from an invisible
	// team. Check resource visibility only in the empty-result case.
	if len(members) == 0 {
		if _, err := GetTeam(ctx, store, actorAccountID, teamID); err != nil {
			return nil, err
		}
	}
	return teamMembersFromDB(members), nil
}

// RemoveTeamUser retains the standalone transaction contract for model callers.
// API writes use the service-owned transaction to include session authority.
func RemoveTeamUser(ctx context.Context, store db.Store, actorAccountID, teamID, personID int64) error {
	return db.WithTx(db.WithStore(ctx, store), func(txCtx context.Context) error {
		q, err := db.TxQueries(txCtx)
		if err != nil {
			return err
		}
		return RemoveTeamUserTx(txCtx, q, actorAccountID, teamID, personID)
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
