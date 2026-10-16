package db

import (
	"testing"
	"time"

	"github.com/datvtph41107/bdspro/util"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
)

func TestSessionLifecycle(t *testing.T) {
	ctx, tx, q := beginTestTx(t)
	_, accountID := createRandomAccount(t, ctx, q)
	tokenHash := "fixture-session-" + util.RandomString(32)

	before := time.Now()
	expiresAt, err := q.CreateLoginSession(ctx, CreateLoginSessionParams{
		AccountID: accountID, TokenHash: tokenHash,
	})
	require.NoError(t, err)
	require.WithinDuration(t, before.Add(24*time.Hour), expiresAt, 3*time.Second)

	gotAccountID, err := q.GetSessionAccount(ctx, tokenHash)
	require.NoError(t, err)
	require.Equal(t, accountID, gotAccountID)

	require.NoError(t, q.RevokeSession(ctx, tokenHash))
	_, err = q.GetSessionAccount(ctx, tokenHash)
	require.ErrorIs(t, err, pgx.ErrNoRows)

	// Logical revocation preserves the row for audit/lifecycle purposes.
	var revokedAt time.Time
	err = tx.QueryRow(ctx, `SELECT revoked_at FROM sessions WHERE token_hash=$1`, tokenHash).Scan(&revokedAt)
	require.NoError(t, err)
	require.False(t, revokedAt.IsZero())

	// RevokeSession is idempotent (:exec; no affected rows required).
	require.NoError(t, q.RevokeSession(ctx, tokenHash))
}

func TestExpiredSession(t *testing.T) {
	ctx, tx, q := beginTestTx(t)
	_, accountID := createRandomAccount(t, ctx, q)
	tokenHash := "fixture-expired-session-" + util.RandomString(32)
	_, err := q.CreateLoginSession(ctx, CreateLoginSessionParams{
		AccountID: accountID, TokenHash: tokenHash,
	})
	require.NoError(t, err)

	// Keep CHECK(expires_at > created_at) valid while simulating expiry.
	_, err = tx.Exec(ctx, `
		UPDATE sessions
		SET created_at=now()-interval '48 hours',
		    expires_at=now()-interval '24 hours'
		WHERE token_hash=$1`, tokenHash)
	require.NoError(t, err)

	_, err = q.GetSessionAccount(ctx, tokenHash)
	require.ErrorIs(t, err, pgx.ErrNoRows)
}

func TestClosedAccountSession(t *testing.T) {
	ctx, tx, q := beginTestTx(t)
	_, accountID := createRandomAccount(t, ctx, q)
	tokenHash := "fixture-closed-session-" + util.RandomString(32)
	_, err := q.CreateLoginSession(ctx, CreateLoginSessionParams{
		AccountID: accountID, TokenHash: tokenHash,
	})
	require.NoError(t, err)

	_, err = tx.Exec(ctx, `UPDATE accounts SET closed_at=now() WHERE id=$1`, accountID)
	require.NoError(t, err)
	_, err = q.GetSessionAccount(ctx, tokenHash)
	require.ErrorIs(t, err, pgx.ErrNoRows)
}

func TestSessionTokenHashUnique(t *testing.T) {
	ctx, _, q := beginTestTx(t)
	_, accountID := createRandomAccount(t, ctx, q)
	tokenHash := "fixture-duplicate-" + util.RandomString(32)
	_, err := q.CreateLoginSession(ctx, CreateLoginSessionParams{
		AccountID: accountID, TokenHash: tokenHash,
	})
	require.NoError(t, err)

	_, err = q.CreateLoginSession(ctx, CreateLoginSessionParams{
		AccountID: accountID, TokenHash: tokenHash,
	})
	var pgErr *pgconn.PgError
	require.ErrorAs(t, err, &pgErr)
	require.Equal(t, "23505", pgErr.Code)
}
