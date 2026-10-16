package db

import (
	"context"
	"strings"
	"testing"

	"github.com/datvtph41107/bdspro/util"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
)

// Create a valid Party -> Person -> Account fixture in the caller's
// transaction. The caller controls rollback and isolation.
func createRandomAccount(t *testing.T, ctx context.Context, q *Queries) (personID, accountID int64) {
	t.Helper()
	personID, err := q.CreatePersonParty(ctx)
	require.NoError(t, err)
	require.Positive(t, personID)

	require.NoError(t, q.CreatePersonSubtype(ctx, personID))
	accountID, err = q.CreateAccount(ctx, personID)
	require.NoError(t, err)
	require.Positive(t, accountID)
	return personID, accountID
}

// Verified email fixture for Organization queries. No real password is needed.
func createVerifiedAccount(t *testing.T, ctx context.Context, q *Queries) (personID, accountID int64, email string) {
	t.Helper()
	personID, accountID = createRandomAccount(t, ctx, q)
	email = util.RandomEmail()
	emailID, err := q.CreatePrimaryAccountEmail(ctx, CreatePrimaryAccountEmailParams{
		AccountID: accountID,
		Email:     email,
	})
	require.NoError(t, err)
	require.Positive(t, emailID)
	require.NoError(t, q.MarkAccountEmailVerified(ctx, emailID))
	return personID, accountID, email
}

func TestCreateAccount(t *testing.T) {
	ctx, tx, q := beginTestTx(t)
	personID, accountID := createRandomAccount(t, ctx, q)

	actual, err := q.GetOpenAccountPersonID(ctx, accountID)
	require.NoError(t, err)
	require.Equal(t, personID, actual)

	// Rollback explicitly, then verify the data is not visible from the pool.
	require.NoError(t, tx.Rollback(ctx))
	_, err = New(testDB).GetInvitationAccountPersonID(ctx, accountID)
	require.ErrorIs(t, err, pgx.ErrNoRows)
}

func TestCreateAccountInvalidPerson(t *testing.T) {
	ctx, _, q := beginTestTx(t)
	_, err := q.CreateAccount(ctx, -1)
	var pgErr *pgconn.PgError
	require.ErrorAs(t, err, &pgErr)
	require.Equal(t, "23503", pgErr.Code)
}

func TestCreatePrimaryAccountEmail(t *testing.T) {
	ctx, tx, q := beginTestTx(t)
	_, accountID := createRandomAccount(t, ctx, q)
	email := util.RandomEmail()

	emailID, err := q.CreatePrimaryAccountEmail(ctx, CreatePrimaryAccountEmailParams{
		AccountID: accountID, Email: email,
	})
	require.NoError(t, err)
	require.Positive(t, emailID)

	var savedEmail string
	var isPrimary bool
	err = tx.QueryRow(ctx, `SELECT email, is_primary FROM account_emails WHERE id=$1`, emailID).
		Scan(&savedEmail, &isPrimary)
	require.NoError(t, err)
	require.Equal(t, email, savedEmail)
	require.True(t, isPrimary)
}

func TestPrimaryEmailCaseInsensitiveUnique(t *testing.T) {
	ctx, _, q := beginTestTx(t)
	_, accountA := createRandomAccount(t, ctx, q)
	_, accountB := createRandomAccount(t, ctx, q)
	email := util.RandomEmail()

	_, err := q.CreatePrimaryAccountEmail(ctx, CreatePrimaryAccountEmailParams{
		AccountID: accountA, Email: email,
	})
	require.NoError(t, err)

	// Different Account, same login key ignoring email case.
	_, err = q.CreatePrimaryAccountEmail(ctx, CreatePrimaryAccountEmailParams{
		AccountID: accountB, Email: strings.ToUpper(email),
	})
	var pgErr *pgconn.PgError
	require.ErrorAs(t, err, &pgErr)
	require.Equal(t, "23505", pgErr.Code)
}

func TestGetLoginCredentials(t *testing.T) {
	ctx, _, q := beginTestTx(t)
	_, accountID := createRandomAccount(t, ctx, q)
	email := util.RandomEmail()
	passwordHash := "fixture-hash-not-for-login"

	_, err := q.CreatePrimaryAccountEmail(ctx, CreatePrimaryAccountEmailParams{
		AccountID: accountID, Email: strings.ToUpper(email),
	})
	require.NoError(t, err)
	require.NoError(t, q.CreatePasswordCredential(ctx, CreatePasswordCredentialParams{
		AccountID: accountID, PasswordHash: passwordHash,
	}))

	// Query uses lower(btrim(stored_email)) = $1.
	credentials, err := q.GetLoginCredentials(ctx, email)
	require.NoError(t, err)
	require.Equal(t, accountID, credentials.AccountID)
	require.Equal(t, passwordHash, credentials.PasswordHash)
	require.False(t, credentials.VerifiedAt.Valid)

	_, err = q.GetLoginCredentials(ctx, "missing-"+util.RandomEmail())
	require.ErrorIs(t, err, pgx.ErrNoRows)
}

func TestPasswordCredentialOnePerAccount(t *testing.T) {
	ctx, _, q := beginTestTx(t)
	_, accountID := createRandomAccount(t, ctx, q)
	require.NoError(t, q.CreatePasswordCredential(ctx, CreatePasswordCredentialParams{
		AccountID: accountID, PasswordHash: "fixture-one",
	}))

	err := q.CreatePasswordCredential(ctx, CreatePasswordCredentialParams{
		AccountID: accountID, PasswordHash: "fixture-two",
	})
	var pgErr *pgconn.PgError
	require.ErrorAs(t, err, &pgErr)
	require.Equal(t, "23505", pgErr.Code)
}

func TestClosedAccountNotOpen(t *testing.T) {
	ctx, tx, q := beginTestTx(t)
	personID, accountID := createRandomAccount(t, ctx, q)

	_, err := tx.Exec(ctx, `UPDATE accounts SET closed_at=now() WHERE id=$1`, accountID)
	require.NoError(t, err)

	_, err = q.GetOpenAccountPersonID(ctx, accountID)
	require.ErrorIs(t, err, pgx.ErrNoRows)

	// Invitation-specific query intentionally doesn't apply the closed filter.
	gotPersonID, err := q.GetInvitationAccountPersonID(ctx, accountID)
	require.NoError(t, err)
	require.Equal(t, personID, gotPersonID)
}
