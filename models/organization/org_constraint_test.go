package organization

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

// SQLSTATE alone cannot identify which business invariant was violated.
func TestAddMemberClassifiesOnlyKnownConstraints(t *testing.T) {
	store := &organizationStoreFake{}
	check := func(t *testing.T, code, constraint string, want error) {
		t.Helper()
		cause := &pgconn.PgError{Code: code, ConstraintName: constraint, Message: "private database credential"}
		store.memberErr = cause
		_, err := AddOrgUser(context.Background(), store, 11, 23, 31)
		if err == nil || !errors.Is(err, cause) {
			t.Fatalf("driver cause not retained: %v", err)
		}
		if want != nil && err.Error() != want.Error() {
			t.Fatalf("model error exposed internal database detail: %q", err.Error())
		}
		if want != nil && !errors.Is(err, want) {
			t.Fatalf("constraint %s: got %v, want %v", constraint, err, want)
		}
		if want == nil && (errors.Is(err, ErrMembershipExists) ||
			errors.Is(err, ErrPersonUnavailable)) {
			t.Fatalf("unrelated constraint falsely classified: %v", err)
		}
	}

	t.Run("active membership unique index", func(t *testing.T) {
		check(t, "23505", "organization_members_one_open_uq", ErrMembershipExists)
	})
	t.Run("person foreign key", func(t *testing.T) {
		check(t, "23503", "organization_members_person_id_fkey", ErrPersonUnavailable)
	})
	t.Run("unrelated unique violation", func(t *testing.T) {
		check(t, "23505", "other_unique_constraint", nil)
	})
	t.Run("organization foreign key", func(t *testing.T) {
		check(t, "23503", "organization_members_organization_id_fkey", nil)
	})
	t.Run("unknown PostgreSQL code", func(t *testing.T) {
		check(t, "23514", "organization_members_role_check", nil)
	})
}
