package db

import (
	"testing"

	"github.com/datvtph41107/bdspro/util"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

func TestEmailVerification(t *testing.T) {
	ctx, _, q := beginTestTx(t)
	_, accountID := createRandomAccount(t, ctx, q)
	email := util.RandomEmail()
	tokenHash := "fixture-email-token-" + util.RandomString(32)

	emailID, err := q.CreatePrimaryAccountEmail(ctx, CreatePrimaryAccountEmailParams{
		AccountID: accountID, Email: email,
	})
	require.NoError(t, err)
	require.NoError(t, q.CreatePasswordCredential(ctx, CreatePasswordCredentialParams{
		AccountID: accountID, PasswordHash: "fixture-hash-not-for-login",
	}))

	before, err := q.GetLoginCredentials(ctx, email)
	require.NoError(t, err)
	require.False(t, before.VerifiedAt.Valid)

	require.NoError(t, q.CreateEmailVerificationToken(ctx, CreateEmailVerificationTokenParams{
		AccountEmailID: emailID, TokenHash: tokenHash,
	}))

	matchedID, err := q.ConsumeEmailVerificationToken(ctx, tokenHash)
	require.NoError(t, err)
	require.Equal(t, emailID, matchedID)
	require.NoError(t, q.MarkAccountEmailVerified(ctx, matchedID))

	after, err := q.GetLoginCredentials(ctx, email)
	require.NoError(t, err)
	require.True(t, after.VerifiedAt.Valid)
	require.False(t, after.VerifiedAt.Time.IsZero())

	// Same token cannot be consumed twice, despite its row still existing.
	_, err = q.ConsumeEmailVerificationToken(ctx, tokenHash)
	require.ErrorIs(t, err, pgx.ErrNoRows)
}

func TestEmailVerificationExpiredToken(t *testing.T) {
	ctx, tx, q := beginTestTx(t)
	_, accountID := createRandomAccount(t, ctx, q)
	emailID, err := q.CreatePrimaryAccountEmail(ctx, CreatePrimaryAccountEmailParams{
		AccountID: accountID, Email: util.RandomEmail(),
	})
	require.NoError(t, err)

	tokenHash := "fixture-expired-" + util.RandomString(32)
	require.NoError(t, q.CreateEmailVerificationToken(ctx, CreateEmailVerificationTokenParams{
		AccountEmailID: emailID, TokenHash: tokenHash,
	}))

	// Test-only clock fixture. Both timestamps must move backwards to satisfy
	// the table CHECK(expires_at > created_at) constraint.
	_, err = tx.Exec(ctx, `
		UPDATE email_verification_tokens
		SET created_at=now()-interval '2 hours',
		    expires_at=now()-interval '1 hour'
		WHERE token_hash=$1`, tokenHash)
	require.NoError(t, err)

	_, err = q.ConsumeEmailVerificationToken(ctx, tokenHash)
	require.ErrorIs(t, err, pgx.ErrNoRows)
}

func TestEmailVerificationUnknownToken(t *testing.T) {
	ctx, _, q := beginTestTx(t)
	_, err := q.ConsumeEmailVerificationToken(ctx, "does-not-exist-"+util.RandomString(20))
	require.ErrorIs(t, err, pgx.ErrNoRows)
}
