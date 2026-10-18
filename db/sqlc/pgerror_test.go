package db

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
)

func TestPostgresErrorUnwrap(t *testing.T) {
	original := &pgconn.PgError{
		Code: "23505", TableName: "account_emails",
		ConstraintName: "account_emails_login_key_uq",
	}
	require.Same(t, original, PostgresError(original))
	require.Same(t, original, PostgresError(fmt.Errorf("registration: %w", original)))
	require.Same(t, original, PostgresError(errors.Join(
		fmt.Errorf("commit: %w", original), errors.New("rollback failed"),
	)))
	for _, err := range []error{
		nil, context.Canceled, context.DeadlineExceeded, pgx.ErrNoRows,
		errors.New("SQLSTATE 23505 is not a typed PostgreSQL error"),
	} {
		require.Nil(t, PostgresError(err), "must not guess SQLSTATE from %v", err)
	}
}

func TestPostgresErrorRealConstraints(t *testing.T) {
	requireCommitTestDB(t)
	ctx := context.Background()

	// Existing permissions are inserted in migration 000003.
	_, err := testDB.Exec(ctx, "INSERT INTO permissions(code) VALUES ('members.read')")
	unique := PostgresError(err)
	require.NotNil(t, unique)
	require.Equal(t, "23505", unique.Code)
	require.Equal(t, "permissions", unique.TableName)
	require.Equal(t, "permissions_pkey", unique.ConstraintName)

	// A foreign-key failure must not be treated as the same conflict.
	_, err = testDB.Exec(ctx,
		"INSERT INTO organization_roles(organization_id,name) VALUES (-1,'missing organization')")
	fk := PostgresError(err)
	require.NotNil(t, fk)
	require.Equal(t, "23503", fk.Code)
	require.NotEqual(t, unique.ConstraintName, fk.ConstraintName)

	// Request cancellation has no SQLSTATE guarantee and is not a retry hint.
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	_, err = testDB.Exec(canceled, "SELECT 1")
	require.ErrorIs(t, err, context.Canceled)
	require.Nil(t, PostgresError(err))
}
