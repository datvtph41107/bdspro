package db

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestIsUniqueViolation(t *testing.T) {
	const constraint = "team_members_pkey"
	var typedNil *pgconn.PgError
	unique := &pgconn.PgError{Code: "23505", ConstraintName: constraint}
	for _, tc := range []struct {
		name       string
		err        error
		constraint string
		want       bool
	}{
		{"exact", unique, constraint, true},
		{"wrapped", fmt.Errorf("insert: %w", unique), constraint, true},
		{"wrong constraint", unique, "teams_pkey", false},
		{"wrong code", &pgconn.PgError{Code: "23503", ConstraintName: constraint}, constraint, false},
		{"unnamed", &pgconn.PgError{Code: "23505"}, "", false},
		{"empty expected", unique, "", false},
		{"ordinary error", errors.New("23505"), constraint, false},
		{"nil", nil, constraint, false},
		{"typed nil", typedNil, constraint, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsUniqueViolation(tc.err, tc.constraint); got != tc.want {
				t.Fatalf("IsUniqueViolation = %v, want %v", got, tc.want)
			}
		})
	}
}
