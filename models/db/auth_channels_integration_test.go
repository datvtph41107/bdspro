package db

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

func authChannelsTestStore(t *testing.T) (context.Context, *pgxpool.Pool, Store) {
	t.Helper()
	url := os.Getenv("BDS_TEST_DB_URL")
	if url == "" {
		t.Skip("requires disposable bdspro_test database")
	}
	if os.Getenv("BDS_TEST_MODE") != "disposable" {
		t.Fatal("refusing database writes outside disposable mode")
	}
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ConnConfig.Database != "bdspro_test" {
		t.Fatal("refusing to write to application database")
	}
	ctx := context.Background()
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if err := pool.Ping(ctx); err != nil {
		t.Fatal(err)
	}
	return ctx, pool, NewStore(pool)
}

func TestPhoneSignupAtomicityUniquenessAndWrongAttempt(t *testing.T) {
	ctx, pool, store := authChannelsTestStore(t)
	phone := fmt.Sprintf("+84%09d", time.Now().UnixNano()%1000000000)
	mac := sha256.Sum256([]byte("valid code"))
	wrong := sha256.Sum256([]byte("incorrect"))
	if err := store.StartPhoneSignup(ctx, phone, mac[:], "123456"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CompletePhoneSignup(ctx, phone, wrong[:]); !errors.Is(err, ErrOTPInvalid) {
		t.Fatalf("wrong OTP: %v", err)
	}
	var attempts int
	if err := pool.QueryRow(ctx, "SELECT failed_attempts FROM phone_signup_challenges WHERE phone_e164=$1", phone).Scan(&attempts); err != nil {
		t.Fatal(err)
	}
	if attempts != 1 {
		t.Fatalf("wrong OTP failure not committed: %d", attempts)
	}
	account, err := store.CompletePhoneSignup(ctx, phone, mac[:])
	if err != nil {
		t.Fatal(err)
	}
	if account.ID <= 0 || account.Email != "" {
		t.Fatalf("unexpected phone-only account: %+v", account)
	}
	owner, err := store.AccountIDForPhone(ctx, phone)
	if err != nil || owner != account.ID {
		t.Fatalf("phone owner=%d err=%v", owner, err)
	}
	if _, err := store.GetPasswordHash(ctx, account.ID); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("phone-only account should be passwordless: %v", err)
	}
	if err := store.StartPhoneSignup(ctx, phone, mac[:], "123456"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CompletePhoneSignup(ctx, phone, mac[:]); err == nil {
		t.Fatal("same phone registered twice")
	} else {
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || pgErr.Code != "23505" {
			t.Fatalf("expected phone uniqueness conflict: %v", err)
		}
	}
	// Assert only the affected phone identity: other test packages may
	// legitimately create Parties concurrently in the same disposable DB.
	var credentials int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM phone_credentials WHERE phone_e164=$1", phone).Scan(&credentials); err != nil {
		t.Fatal(err)
	}
	if credentials != 1 {
		t.Fatalf("duplicate registration changed phone ownership: count=%d", credentials)
	}
	currentOwner, err := store.AccountIDForPhone(ctx, phone)
	if err != nil || currentOwner != owner {
		t.Fatalf("duplicate registration changed account owner: owner=%d error=%v", currentOwner, err)
	}
}

func TestPasswordResetOnceAndCredentialCreation(t *testing.T) {
	ctx, _, store := authChannelsTestStore(t)
	account, err := store.RegisterTx(ctx, RegisterTxParams{Email: uuid.NewString() + "@example.test", PasswordHash: "initial-test-hash"})
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256([]byte(uuid.NewString()))
	if err := store.CreatePasswordResetToken(ctx, CreatePasswordResetTokenParams{TokenHash: digest[:], AccountID: account.ID}); err != nil {
		t.Fatal(err)
	}
	if err := store.ResetPasswordTx(ctx, digest[:], "replacement-test-hash"); err != nil {
		t.Fatal(err)
	}
	hash, err := store.GetPasswordHash(ctx, account.ID)
	if err != nil || hash != "replacement-test-hash" {
		t.Fatalf("reset did not persist credential: hash=%q err=%v", hash, err)
	}
	if err := store.ResetPasswordTx(ctx, digest[:], "third-test-hash"); !errors.Is(err, ErrPasswordResetInvalid) {
		t.Fatalf("reset token replay: %v", err)
	}
}
