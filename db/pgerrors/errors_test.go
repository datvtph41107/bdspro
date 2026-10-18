package pgerrors

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestUniqueConstraintRecognizesOnlyKnownInvariant(t *testing.T) {
	tests := []struct {
		name string
		err error
		matched bool
	}{
		{"direct", &pgconn.PgError{Code: UniqueViolation, ConstraintName: "account_emails_login_key_uq"}, true},
		{"wrapped", fmt.Errorf("register: %w", &pgconn.PgError{Code: UniqueViolation, ConstraintName: "account_emails_login_key_uq"}), true},
		{"joined", errors.Join(context.Canceled, &pgconn.PgError{Code: UniqueViolation, ConstraintName: "account_emails_login_key_uq"}), true},
		{"unknown constraint", &pgconn.PgError{Code: UniqueViolation, ConstraintName: "other_table_key"}, false},
		{"foreign key same name", &pgconn.PgError{Code: ForeignKeyViolation, ConstraintName: "account_emails_login_key_uq"}, false},
		{"no constraint", &pgconn.PgError{Code: UniqueViolation}, false},
		{"transport", context.DeadlineExceeded, false},
		{"nil", nil, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := UniqueConstraint(tc.err, "account_emails_login_key_uq"); got != tc.matched {
				t.Fatalf("UniqueConstraint()=%v, want %v", got, tc.matched)
			}
		})
	}
}

func TestTransientSQLSTATEClassificationIsNotAnAutomaticRetry(t *testing.T) {
	for _, tc := range []struct {
		code string
		retryable bool
	}{
		{SerializationFailure, true},
		{DeadlockDetected, true},
		{LockNotAvailable, false},
		{UniqueViolation, false},
		{ForeignKeyViolation, false},
		{CheckViolation, false},
	} {
		err := fmt.Errorf("tx failed: %w", &pgconn.PgError{Code: tc.code})
		if got := RetryableTransactionFailure(err); got != tc.retryable {
			t.Errorf("code %s: retryable=%v, want %v", tc.code, got, tc.retryable)
		}
	}
	if RetryableTransactionFailure(context.Canceled) || RetryableTransactionFailure(nil) {
		t.Fatal("context cancellation or nil cannot be classified as serialization/deadlock")
	}
}
