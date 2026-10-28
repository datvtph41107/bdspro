package db

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// This test runs only against the disposable database provisioned by CI.
// All inserted rows are rolled back to leave no state for later tests.
func TestPhoneBindingDoesNotChangeExistingPhoneOrStealNumber(t *testing.T) {
	if os.Getenv("BDS_TEST_MODE") != "disposable" {
		t.Skip("requires disposable test database")
	}
	url := os.Getenv("BDS_TEST_DB_URL")
	if url == "" {
		t.Fatal("BDS_TEST_DB_URL is required")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	q := New(tx)
	createAccount := func() int64 {
		partyID, err := q.CreateParty(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if err := q.CreatePerson(ctx, partyID); err != nil {
			t.Fatal(err)
		}
		a, err := q.CreateAccount(ctx, CreateAccountParams{
			PersonID: partyID, Email: uuid.NewString() + "@example.test",
		})
		if err != nil {
			t.Fatal(err)
		}
		return a.ID
	}
	first, second := createAccount(), createAccount()
	firstPhone, replacement := "+84912340001", "+84912340002"
	bind := func(account int64, phone string) error {
		return q.BindPhoneCredentialQuery(ctx, BindPhoneCredentialQueryParams{
			AccountID: account, PhoneE164: phone,
		})
	}
	if err := bind(first, firstPhone); err != nil {
		t.Fatal(err)
	}
	// Use savepoints so PostgreSQL uniqueness errors do not poison the
	// outer test transaction.
	for _, tc := range []struct {
		account int64
		phone   string
	}{
		{first, replacement},
		{second, firstPhone},
	} {
		if _, err := tx.Exec(ctx, "SAVEPOINT phone_conflict"); err != nil {
			t.Fatal(err)
		}
		err := bind(tc.account, tc.phone)
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || pgErr.Code != "23505" {
			t.Fatalf("expected unique phone constraint violation, got %v", err)
		}
		if _, err := tx.Exec(ctx, "ROLLBACK TO SAVEPOINT phone_conflict"); err != nil {
			t.Fatal(err)
		}
		if _, err := tx.Exec(ctx, "RELEASE SAVEPOINT phone_conflict"); err != nil {
			t.Fatal(err)
		}
	}
	stored, err := q.GetPhoneAccountIDQuery(ctx, firstPhone)
	if err != nil || stored != first {
		t.Fatalf("original phone owner changed: owner=%d err=%v", stored, err)
	}
	if _, err := q.GetPhoneAccountIDQuery(ctx, replacement); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("replacement number unexpectedly linked: %v", err)
	}
}
