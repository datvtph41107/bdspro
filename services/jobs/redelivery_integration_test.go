package jobs

import (
	"context"
	"crypto/sha256"
	"errors"
	"os"
	"regexp"
	"testing"
	"time"

	account_model "github.com/datvtph41107/bdspro/models/account"
	auth_model "github.com/datvtph41107/bdspro/models/auth"
	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// A mail provider can accept the reset message and still report an error to
// the worker. Retrying creates another independent proof today. Verify the
// actual database recovery transaction revokes BOTH after either is redeemed.
func TestPasswordResetRedeliveryCannotLeaveAnotherUsableProofAfterRecovery(t *testing.T) {
	source := os.Getenv("BDS_TEST_DB_URL")
	if source == "" {
		t.Skip("requires migrated disposable PostgreSQL")
	}
	if os.Getenv("BDS_TEST_MODE") != "disposable" {
		t.Fatal("refusing writes without disposable mode")
	}
	cfg, err := pgxpool.ParseConfig(source)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ConnConfig.Database != "bdspro_test" {
		t.Fatal("refusing writes outside bdspro_test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
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
	account, err := account_model.Register(ctx, store, account_model.RegisterParams{
		Email: uuid.NewString() + "@example.test", PasswordHash: "initial-test-hash",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.MarkEmailVerified(ctx, db.MarkEmailVerifiedParams{
		ID: account.ID, Email: account.Email,
	}); err != nil {
		t.Fatal(err)
	}
	mailer := &ambiguousAcceptanceMailer{}
	if err := SendPasswordResetEmail(ctx, store, mailer, "https://example.test", account.ID); err == nil {
		t.Fatal("ambiguous SMTP acceptance must remain retryable")
	}
	if err := SendPasswordResetEmail(ctx, store, mailer, "https://example.test", account.ID); err != nil {
		t.Fatal(err)
	}
	if len(mailer.messages) != 2 {
		t.Fatalf("expected two SMTP acceptance attempts, got %d", len(mailer.messages))
	}
	proofs := make([][]byte, 0, 2)
	for _, body := range mailer.messages {
		match := regexp.MustCompile("token=([A-Za-z0-9_-]{43})").FindStringSubmatch(body)
		if len(match) != 2 {
			t.Fatal("missing password reset link")
		}
		digest := sha256.Sum256([]byte(match[1]))
		proofs = append(proofs, digest[:])
		var exists bool
		if err := pool.QueryRow(ctx,
			"SELECT EXISTS(SELECT 1 FROM password_reset_tokens WHERE account_id=$1 AND token_hash=$2)",
			account.ID, digest[:]).Scan(&exists); err != nil || !exists {
			t.Fatalf("SMTP proof was not durably recorded: exists=%t err=%v", exists, err)
		}
	}
	if string(proofs[0]) == string(proofs[1]) {
		t.Fatal("retry unexpectedly reused an old reset proof; update the explicit contract")
	}
	if err := auth_model.ResetPassword(ctx, store, proofs[0], "replacement-test-hash"); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := pool.QueryRow(ctx,
		"SELECT count(*) FROM password_reset_tokens WHERE account_id=$1",
		account.ID).Scan(&count); err != nil || count != 0 {
		t.Fatalf("recovery did not revoke all reset proofs: count=%d err=%v", count, err)
	}
	for _, proof := range proofs {
		if err := auth_model.ResetPassword(ctx, store, proof, "again-test-hash"); !errors.Is(err, auth_model.ErrPasswordResetInvalid) {
			t.Fatalf("previously issued proof remained usable after recovery: %v", err)
		}
	}
}
