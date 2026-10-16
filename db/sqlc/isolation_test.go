package db

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

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
