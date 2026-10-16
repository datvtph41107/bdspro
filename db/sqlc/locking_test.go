package db

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

// Lock rows in a global ascending order, regardless of caller input.
// This is a test demonstration of Lecture #7-8, not an unused CRUD query.
func lockAccountsInOrder(ctx context.Context, tx pgx.Tx, firstID, secondID int64) error {
	ids := []int64{firstID, secondID}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	for i, id := range ids {
		var lockedID int64
		if err := tx.QueryRow(ctx,
			"SELECT id FROM accounts WHERE id=$1 FOR UPDATE", id,
		).Scan(&lockedID); err != nil {
			return err
		}
		if i == 0 {
			// Give competing transactions an opportunity to acquire locks.
			time.Sleep(5 * time.Millisecond)
		}
	}
	return nil
}

func TestConcurrentLocksFollowGlobalOrder(t *testing.T) {
	requireCommitTestDB(t)
	first := createCommittedRegistration(t)
	second := createCommittedRegistration(t)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	const workers = 6
	results := make(chan error, workers)

	for i := 0; i < workers; i++ {
		reversed := i%2 == 1
		go func() {
			a, b := first.result.AccountID, second.result.AccountID
			if reversed {
				a, b = b, a
			}
			err := pgx.BeginFunc(ctx, testDB, func(tx pgx.Tx) error {
				return lockAccountsInOrder(ctx, tx, a, b)
			})
			results <- err
		}()
	}
	for i := 0; i < workers; i++ {
		require.NoError(t, <-results)
	}
	// Without global ordering, acquiring two row locks in opposite orders
	// may deadlock. This test asserts the safe path, not deadlock occurrence.
}
