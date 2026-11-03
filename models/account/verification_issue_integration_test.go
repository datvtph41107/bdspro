package account_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	account_model "github.com/datvtph41107/bdspro/models/account"
	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Verifies real database locking, idempotent retries and the checked
// transition to an already verified account, without mocking SQLC queries.
func TestIssueVerificationReusesCodeAndStopsAfterVerification(t *testing.T) {
	source := os.Getenv("BDS_TEST_DB_URL")
	if source == "" {
		t.Skip("requires disposable bdspro_test database")
	}
	if os.Getenv("BDS_TEST_MODE") != "disposable" {
		t.Fatal("refusing database writes outside disposable mode")
	}
	cfg, err := pgxpool.ParseConfig(source)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ConnConfig.Database != "bdspro_test" {
		t.Fatal("refusing writes outside bdspro_test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		t.Fatal(err)
	}
	store := db.NewStore(pool)
	owner, err := account_model.Register(ctx, store, account_model.RegisterParams{
		Email: uuid.NewString() + "@example.test", PasswordHash: "test-only",
	})
	if err != nil {
		t.Fatal(err)
	}
	first, err := account_model.GetOrCreateVerifyEmail(ctx, store, db.CreateVerifyEmailParams{
		AccountID: owner.ID, Email: owner.Email, SecretCode: "first-" + uuid.NewString(),
	})
	if err != nil {
		t.Fatal(err)
	}
	second, err := account_model.GetOrCreateVerifyEmail(ctx, store, db.CreateVerifyEmailParams{
		AccountID: owner.ID, Email: owner.Email, SecretCode: "new-" + uuid.NewString(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != second.ID || first.SecretCode != second.SecretCode {
		t.Fatalf("retry must reuse persisted proof: first=%+v second=%+v", first, second)
	}
	if _, err := account_model.VerifyEmail(ctx, store, db.ConsumeVerifyEmailParams{
		ID: first.ID, SecretCode: first.SecretCode,
	}); err != nil {
		t.Fatal(err)
	}
	_, err = account_model.GetOrCreateVerifyEmail(ctx, store, db.CreateVerifyEmailParams{
		AccountID: owner.ID, Email: owner.Email, SecretCode: "unused-" + uuid.NewString(),
	})
	if !errors.Is(err, account_model.ErrEmailAlreadyVerified) {
		t.Fatalf("already verified account must not issue more codes: %v", err)
	}
}
