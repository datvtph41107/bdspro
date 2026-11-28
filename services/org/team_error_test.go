package org

import (
	"errors"
	"fmt"
	"testing"

	"github.com/datvtph41107/bdspro/models/organization"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestMapTeamWriteError(t *testing.T) {
	for _, tc := range []struct {
		name       string
		code       string
		constraint string
		conflict   bool
	}{
		{"team name", "23505", "teams_org_name_uq", true},
		{"primary key", "23505", "teams_pkey", false},
		{"other unique", "23505", "team_invites_open_uq", false},
		{"foreign key", "23503", "teams_organization_id_fkey", false},
		{"wrong SQLSTATE", "23514", "teams_org_name_uq", false},
		{"missing constraint", "23505", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := fmt.Errorf("create team: %w", &pgconn.PgError{Code: tc.code, ConstraintName: tc.constraint})
			got := mapTeamWriteError(err)
			if tc.conflict {
				if !errors.Is(got, organization.ErrTeamExists) {
					t.Fatalf("expected team name conflict, got %v", got)
				}
			} else if got != err {
				t.Fatalf("unexpected error replacement: %v", got)
			}
		})
	}
	if mapTeamWriteError(nil) != nil {
		t.Fatal("nil must remain nil")
	}
	err := errors.New("connection unavailable")
	if mapTeamWriteError(err) != err {
		t.Fatal("non-PostgreSQL error must remain unchanged")
	}
}
