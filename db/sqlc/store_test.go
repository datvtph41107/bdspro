package db

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/datvtph41107/bdspro/util"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
)

// Commit-based tests must never use the developer's bdspro database.
func requireCommitTestDB(t *testing.T) {
	t.Helper()
	if testDB == nil {
		t.Skip("SQLC_TEST_DATABASE_URL is not configured")
	}
	var name string
	require.NoError(t, testDB.QueryRow(context.Background(), "SELECT current_database()").Scan(&name))
	if name != "bdspro_sqlc_ci" {
		t.Skip("transaction COMMIT tests require disposable bdspro_sqlc_ci")
	}
}

type registrationFixture struct {
	result RegisterTxResult
	email  string
	hash   string
	token  string
}

func createCommittedRegistration(t *testing.T) registrationFixture {
	t.Helper()
	requireCommitTestDB(t)
	ctx := context.Background()
	f := registrationFixture{
		email: util.RandomEmail(),
		hash:  "fixture-password-hash",
		token: "fixture-" + util.RandomString(40),
	}
	var err error
	f.result, err = NewStore(testDB).RegisterTx(ctx, RegisterTxParams{
		Email: f.email, PasswordHash: f.hash, TokenHash: f.token,
	})
	require.NoError(t, err)
	require.Positive(t, f.result.PartyID)
	require.Positive(t, f.result.AccountID)
	require.Positive(t, f.result.AccountEmailID)
	t.Cleanup(func() { cleanupRegistration(t, f.result) })
	return f
}

// Delete only the committed test fixture, using FK dependency order and a
// separate atomic transaction. Never truncate or restart identity sequences.
func cleanupRegistration(t *testing.T, result RegisterTxResult) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err := pgx.BeginFunc(ctx, testDB, func(tx pgx.Tx) error {
		statements := []struct {
			sql string
			id  int64
		}{
			{"DELETE FROM email_verification_tokens WHERE account_email_id=$1", result.AccountEmailID},
			{"DELETE FROM password_credentials WHERE account_id=$1", result.AccountID},
			{"DELETE FROM account_emails WHERE account_id=$1", result.AccountID},
			{"DELETE FROM accounts WHERE id=$1", result.AccountID},
			{"DELETE FROM persons WHERE party_id=$1", result.PartyID},
			{"DELETE FROM parties WHERE id=$1", result.PartyID},
		}
		for _, stmt := range statements {
			if _, err := tx.Exec(ctx, stmt.sql, stmt.id); err != nil {
				return err
			}
		}
		return nil
	})
	require.NoError(t, err)
}

func TestStoreRegisterTxCommit(t *testing.T) {
	f := createCommittedRegistration(t)
	ctx := context.Background()

	// The transaction has returned: a fresh pooled connection sees every row.
	got, err := New(testDB).GetLoginCredentials(ctx, f.email)
	require.NoError(t, err)
	require.Equal(t, f.result.AccountID, got.AccountID)
	require.Equal(t, f.hash, got.PasswordHash)
	require.False(t, got.VerifiedAt.Valid)

	var kind, token string
	err = testDB.QueryRow(ctx, "SELECT kind FROM parties WHERE id=$1", f.result.PartyID).Scan(&kind)
	require.NoError(t, err)
	require.Equal(t, "PERSON", kind)
	err = testDB.QueryRow(ctx,
		"SELECT token_hash FROM email_verification_tokens WHERE account_email_id=$1",
		f.result.AccountEmailID,
	).Scan(&token)
	require.NoError(t, err)
	require.Equal(t, f.token, token)
}

func TestStoreRegisterTxRollback(t *testing.T) {
	f := createCommittedRegistration(t)
	ctx := context.Background()
	var before, after int
	require.NoError(t, testDB.QueryRow(ctx, "SELECT count(*) FROM parties").Scan(&before))

	// The duplicate email fails AFTER Party, Person and Account INSERTs.
	_, err := NewStore(testDB).RegisterTx(ctx, RegisterTxParams{
		Email: strings.ToUpper(f.email), PasswordHash: "duplicate-hash",
		TokenHash: "duplicate-" + util.RandomString(40),
	})
	var pgErr *pgconn.PgError
	require.ErrorAs(t, err, &pgErr)
	require.Equal(t, "23505", pgErr.Code)
	require.NoError(t, testDB.QueryRow(ctx, "SELECT count(*) FROM parties").Scan(&after))
	require.Equal(t, before, after, "partial registration must be rolled back")
}

func TestStoreConcurrentRegistrationUniqueEmail(t *testing.T) {
	requireCommitTestDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	email := util.RandomEmail()
	const workers = 5
	type outcome struct {
		result RegisterTxResult
		err    error
	}
	results := make(chan outcome, workers)
	store := NewStore(testDB)
	for i := 0; i < workers; i++ {
		go func() {
			result, err := store.RegisterTx(ctx, RegisterTxParams{
				Email: email, PasswordHash: "concurrent-fixture-hash",
				TokenHash: "concurrent-" + util.RandomString(40),
			})
			results <- outcome{result, err}
		}()
	}
	successes := 0
	duplicates := 0
	for i := 0; i < workers; i++ {
		out := <-results
		if out.err == nil {
			successes++
			// Register cleanup immediately, including if an assertion later fails.
			result := out.result
			t.Cleanup(func() { cleanupRegistration(t, result) })
			continue
		}
		var pgErr *pgconn.PgError
		if errors.As(out.err, &pgErr) && pgErr.Code == "23505" {
			duplicates++
		} else {
			t.Errorf("unexpected concurrent registration error: %v", out.err)
		}
	}
	require.Equal(t, 1, successes)
	require.Equal(t, workers-1, duplicates)
}

// Lecture #9: in READ COMMITTED a new SELECT sees a concurrent commit;
// in REPEATABLE READ the transaction keeps its original snapshot.
func TestTransactionIsolationReadPhenomena(t *testing.T) {
	requireCommitTestDB(t)
	cases := []struct {
		name           string
		level          pgx.TxIsoLevel
		seesNewVersion bool
	}{
		{"read_committed", pgx.ReadCommitted, true},
		{"repeatable_read", pgx.RepeatableRead, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := createCommittedRegistration(t)
			ctx := context.Background()

			reader, err := testDB.BeginTx(ctx, pgx.TxOptions{IsoLevel: tc.level})
			require.NoError(t, err)
			defer reader.Rollback(context.Background())

			var closedBefore bool
			require.NoError(t, reader.QueryRow(ctx,
				"SELECT closed_at IS NOT NULL FROM accounts WHERE id=$1",
				f.result.AccountID,
			).Scan(&closedBefore))
			require.False(t, closedBefore)

			// A different transaction changes and commits the same row.
			writer, err := testDB.Begin(ctx)
			require.NoError(t, err)
			_, err = writer.Exec(ctx,
				"UPDATE accounts SET closed_at=now() WHERE id=$1",
				f.result.AccountID,
			)
			if err != nil {
				_ = writer.Rollback(ctx)
				t.Fatal(err)
			}
			require.NoError(t, writer.Commit(ctx))

			var closedAfter bool
			require.NoError(t, reader.QueryRow(ctx,
				"SELECT closed_at IS NOT NULL FROM accounts WHERE id=$1",
				f.result.AccountID,
			).Scan(&closedAfter))
			require.Equal(t, tc.seesNewVersion, closedAfter)

			// Roll back the reader before the fixture's Cleanup DELETE.
			require.NoError(t, reader.Rollback(ctx))
		})
	}
}
