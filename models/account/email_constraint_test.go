package account

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestEmailLoginConstraintUsesExactName(t *testing.T) {
	recognized := &pgconn.PgError{
		Code: "23505", ConstraintName: "accounts_email_login_uq",
	}
	if !emailLoginTaken(recognized) || !emailLoginTaken(errors.Join(errors.New("wrapped"), recognized)) {
		t.Fatal("known account-email index not detected")
	}
	for _, e := range []*pgconn.PgError{
		{Code: "23505", ConstraintName: "accounts_pkey"},
		{Code: "23503", ConstraintName: "accounts_email_login_uq"},
	} {
		if emailLoginTaken(e) {
			t.Fatalf("unrelated SQL failure classified as email conflict: %v", e)
		}
	}
}
