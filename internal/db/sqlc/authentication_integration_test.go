package db

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestGoogleSignupIsAtomicAndPasswordless(t *testing.T) {
	url := os.Getenv("BDS_TEST_DB_URL")
	if url == "" {
		t.Skip("requires disposable test database")
	}
	if os.Getenv("BDS_TEST_MODE") != "disposable" {
		t.Fatal("refusing database writes without disposable test mode")
	}
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil { t.Fatal(err) }
	if cfg.ConnConfig.Database != "bdspro_test" {
		t.Fatal("refusing database writes outside bdspro_test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil { t.Fatal(err) }
	defer pool.Close()
	store := NewStore(pool)

	email := uuid.NewString() + "@example.test"
	sub := uuid.NewString()
	account, err := store.CreateGoogleAccountTx(ctx, email, sub)
	if err != nil { t.Fatal(err) }
	if !account.IsEmailVerified || account.Email != email {
		t.Fatalf("invalid Google-only account: %+v", account)
	}
	id, err := store.GetGoogleAccountID(ctx, sub)
	if err != nil || id != account.ID {
		t.Fatalf("Google identity not linked: account=%d link=%d err=%v", account.ID, id, err)
	}
	if _, err := store.GetPasswordHash(ctx, account.ID); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("passwordless account should not have password credential: %v", err)
	}

	var before, after int64
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM parties").Scan(&before); err != nil { t.Fatal(err) }

	// An existing email is never silently linked to a new Google subject.
	_, err = store.CreateGoogleAccountTx(ctx, email, uuid.NewString())
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.ConstraintName != "accounts_email_login_uq" {
		t.Fatalf("duplicate email must be rejected: %v", err)
	}
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM parties").Scan(&after); err != nil { t.Fatal(err) }
	if before != after {
		t.Fatalf("duplicate email created a partial Party: before=%d after=%d", before, after)
	}

	// A Google subject cannot be linked to a second account; the second
	// transaction must not leave behind Party, Person or Account rows.
	_, err = store.CreateGoogleAccountTx(ctx, uuid.NewString()+"@example.test", sub)
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("duplicate subject should reject second account: %v", err)
	}
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM parties").Scan(&after); err != nil { t.Fatal(err) }
	if before != after {
		t.Fatalf("duplicate Google subject created a partial Party: before=%d after=%d", before, after)
	}
}

func TestPasswordRegistrationCreatesCredentialAndOutbox(t *testing.T) {
	url := os.Getenv("BDS_TEST_DB_URL")
	if url == "" { t.Skip("requires disposable test database") }
	if os.Getenv("BDS_TEST_MODE") != "disposable" {
		t.Fatal("refusing database writes without disposable test mode")
	}
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil { t.Fatal(err) }
	if cfg.ConnConfig.Database != "bdspro_test" {
		t.Fatal("refusing database writes outside bdspro_test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil { t.Fatal(err) }
	defer pool.Close()
	store := NewStore(pool)
	account, err := store.RegisterTx(ctx, RegisterTxParams{
		Email: uuid.NewString()+"@example.test", PasswordHash: "test-hash",
	})
	if err != nil { t.Fatal(err) }
	hash, err := store.GetPasswordHash(ctx, account.ID)
	if err != nil || hash != "test-hash" {
		t.Fatalf("password registration lost credential: hash=%q err=%v", hash, err)
	}
	var count int
	if err := pool.QueryRow(ctx,
		"SELECT count(*) FROM job_outbox WHERE task_type = 'task:send_verify_email' AND (payload->>'account_id')::bigint = $1",
		account.ID,
	).Scan(&count); err != nil { t.Fatal(err) }
	if count != 1 { t.Fatalf("registration created %d verification jobs, want 1", count) }
}
