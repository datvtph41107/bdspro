package organization_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"

	db "github.com/datvtph41107/bdspro/models/db"
	organization "github.com/datvtph41107/bdspro/models/organization"
)

func TestListTeamsRejectsUnboundedOptions(t *testing.T) {
	_, err := organization.ListTeams(context.Background(), nil, 1, 2,
		db.ListOptions{Page: 1, PageSize: 20, ListAll: true})
	if !errors.Is(err, organization.ErrTeamPaginationInvalid) {
		t.Fatalf("ListAll should be rejected: %v", err)
	}
}

type teamVisibilityStore struct {
	db.Store

	teams       []db.Team
	members     []db.TeamMember
	listErr     error
	orgErr      error
	teamErr     error
	listTeams   int
	listMembers int
	checkOrg    int
	checkTeam   int
}

func (s *teamVisibilityStore) ListTeamsForAccount(_ context.Context, _ db.ListTeamsForAccountParams) ([]db.Team, error) {
	s.listTeams++
	return s.teams, s.listErr
}

func (s *teamVisibilityStore) ListTeamUsersForAccount(_ context.Context, _ db.ListTeamUsersForAccountParams) ([]db.TeamMember, error) {
	s.listMembers++
	return s.members, s.listErr
}

func (s *teamVisibilityStore) GetOrganizationForAccount(_ context.Context, _ db.GetOrganizationForAccountParams) (db.GetOrganizationForAccountRow, error) {
	s.checkOrg++
	return db.GetOrganizationForAccountRow{PartyID: 71}, s.orgErr
}

func (s *teamVisibilityStore) GetTeamForAccount(_ context.Context, _ db.GetTeamForAccountParams) (db.Team, error) {
	s.checkTeam++
	return db.Team{ID: 91}, s.teamErr
}

// These tests lock in the Gitea-inspired resource visibility boundary without
// paying for a second round-trip on a nonempty SQLC list.
func TestTeamListResourceVisibilityAndQueryBudget(t *testing.T) {
	opts := db.NewListOptions(1, 20, 20, 100)
	databaseErr := errors.New("database offline")
	for _, tt := range []struct {
		name       string
		store      teamVisibilityStore
		wantCount  int
		wantChecks int
		wantErr    error
	}{
		{name: "visible nonempty one query", store: teamVisibilityStore{teams: []db.Team{{ID: 91}}}, wantCount: 1},
		{name: "visible empty two queries", store: teamVisibilityStore{}, wantChecks: 1},
		{name: "hidden empty maps to 404", store: teamVisibilityStore{orgErr: pgx.ErrNoRows}, wantChecks: 1, wantErr: organization.ErrOrganizationNotFound},
		{name: "list failure avoids extra read", store: teamVisibilityStore{listErr: databaseErr}, wantErr: databaseErr},
		{name: "empty visibility lookup failure", store: teamVisibilityStore{orgErr: databaseErr}, wantChecks: 1, wantErr: databaseErr},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := organization.ListTeams(context.Background(), &tt.store, 15, 71, opts)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error=%v want=%v", err, tt.wantErr)
			}
			if len(got) != tt.wantCount {
				t.Fatalf("result count=%d want=%d", len(got), tt.wantCount)
			}
			if tt.store.listTeams != 1 || tt.store.checkOrg != tt.wantChecks {
				t.Fatalf("SQL calls: list=%d check=%d want=1/%d", tt.store.listTeams, tt.store.checkOrg, tt.wantChecks)
			}
		})
	}
}

func TestTeamMemberListResourceVisibilityAndQueryBudget(t *testing.T) {
	opts := db.NewListOptions(1, 20, 20, 100)
	databaseErr := errors.New("database offline")
	for _, tt := range []struct {
		name       string
		store      teamVisibilityStore
		wantCount  int
		wantChecks int
		wantErr    error
	}{
		{name: "visible nonempty one query", store: teamVisibilityStore{members: []db.TeamMember{{TeamID: 91, PersonID: 2}}}, wantCount: 1},
		{name: "visible empty two queries", store: teamVisibilityStore{}, wantChecks: 1},
		{name: "hidden empty maps to 404", store: teamVisibilityStore{teamErr: pgx.ErrNoRows}, wantChecks: 1, wantErr: organization.ErrTeamNotFound},
		{name: "list failure avoids extra read", store: teamVisibilityStore{listErr: databaseErr}, wantErr: databaseErr},
		{name: "empty visibility lookup failure", store: teamVisibilityStore{teamErr: databaseErr}, wantChecks: 1, wantErr: databaseErr},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := organization.ListTeamUsers(context.Background(), &tt.store, 15, 91, opts)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error=%v want=%v", err, tt.wantErr)
			}
			if len(got) != tt.wantCount {
				t.Fatalf("result count=%d want=%d", len(got), tt.wantCount)
			}
			if tt.store.listMembers != 1 || tt.store.checkTeam != tt.wantChecks {
				t.Fatalf("SQL calls: list=%d check=%d want=1/%d", tt.store.listMembers, tt.store.checkTeam, tt.wantChecks)
			}
		})
	}
}
