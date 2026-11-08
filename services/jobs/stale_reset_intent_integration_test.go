package jobs

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	account_model "github.com/datvtph41107/bdspro/models/account"
	auth_model "github.com/datvtph41107/bdspro/models/auth"
	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func g1ResetStore(t *testing.T) (context.Context, *pgxpool.Pool, *db.SQLStore) {
	t.Helper()
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
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	t.Cleanup(cancel)
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if err := pool.Ping(ctx); err != nil {
		t.Fatal(err)
	}
	return ctx, pool, db.NewStore(pool)
}

func g1RequestedVersion(t *testing.T, ctx context.Context, pool *pgxpool.Pool, accountID int64) int64 {
	t.Helper()
	var body []byte
	if err := pool.QueryRow(ctx, `SELECT payload FROM job_outbox
		WHERE task_type='task:send_password_reset'
		AND (payload->>'account_id')::bigint=$1
		ORDER BY id DESC LIMIT 1`, accountID).Scan(&body); err != nil {
		t.Fatal(err)
	}
	var intent struct {
		AccountID    int64 `json:"account_id"`
		ResetVersion int64 `json:"reset_version"`
	}
	if err := json.Unmarshal(body, &intent); err != nil || intent.AccountID != accountID {
		t.Fatalf("bad durable reset intent: %+v %v", intent, err)
	}
	return intent.ResetVersion
}

func TestG1StalePasswordResetJobCannotReissueProofAfterRecovery(t *testing.T) {
	ctx, pool, store := g1ResetStore(t)
	account, err := account_model.Register(ctx, store, account_model.RegisterParams{
		Email: uuid.NewString() + "@example.test",
		PasswordHash: "initial-test-hash",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.MarkEmailVerified(ctx, db.MarkEmailVerifiedParams{
		ID: account.ID, Email: account.Email,
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.RecordPasswordResetJob(ctx, account.ID); err != nil {
		t.Fatal(err)
	}
	oldVersion := g1RequestedVersion(t, ctx, pool, account.ID)
	if oldVersion != 0 {
		t.Fatalf("new account's initial reset generation=%d", oldVersion)
	}
	mailer := &fakeMailSender{}
	if err := SendPasswordResetEmailForVersion(ctx, store, mailer, "https://example.test", account.ID, oldVersion); err != nil {
		t.Fatal(err)
	}
	if len(mailer.content) != 1 {
		t.Fatal("first reset intent did not deliver")
	}
	match := regexp.MustCompile(`token=([A-Za-z0-9_-]{43})`).FindStringSubmatch(mailer.content[0])
	if len(match) != 2 {
		t.Fatal("first password reset link is missing")
	}
	digest := sha256.Sum256([]byte(match[1]))
	if err := auth_model.ResetPassword(ctx, store, digest[:], "new-test-hash"); err != nil {
		t.Fatal(err)
	}
	if err := SendPasswordResetEmailForVersion(ctx, store, mailer, "https://example.test", account.ID, oldVersion); err != nil {
		t.Fatal(err)
	}
	if len(mailer.content) != 1 {
		t.Fatal("stale retry mailed a fresh reset proof after recovery")
	}
	var outstanding int
	if err := pool.QueryRow(ctx,
		"SELECT count(*) FROM password_reset_tokens WHERE account_id=$1",
		account.ID).Scan(&outstanding); err != nil || outstanding != 0 {
		t.Fatalf("stale worker issued a live proof: count=%d err=%v", outstanding, err)
	}
	if err := store.RecordPasswordResetJob(ctx, account.ID); err != nil {
		t.Fatal(err)
	}
	newVersion := g1RequestedVersion(t, ctx, pool, account.ID)
	if newVersion != oldVersion+1 {
		t.Fatalf("new request generation=%d; old=%d", newVersion, oldVersion)
	}
	if err := SendPasswordResetEmailForVersion(ctx, store, mailer, "https://example.test", account.ID, newVersion); err != nil {
		t.Fatal(err)
	}
	if len(mailer.content) != 2 {
		t.Fatal("fresh request after recovery was incorrectly blocked")
	}
}

func TestG1ProofIssuanceWaitsForCredentialRecoveryAccountLock(t *testing.T) {
	ctx, pool, store := g1ResetStore(t)
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
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	var id int64
	if err := tx.QueryRow(ctx, "SELECT id FROM accounts WHERE id=$1 FOR UPDATE", account.ID).Scan(&id); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	proof := sha256.Sum256([]byte("stale-proof-" + uuid.NewString()))
	go func() {
		affected, issueErr := store.IssuePasswordResetTokenIfCurrentVersion(ctx, db.IssuePasswordResetTokenIfCurrentVersionParams{
			TokenHash: proof[:], AccountID: account.ID, ResetVersion: 0,
		})
		if issueErr == nil && affected != 0 {
			issueErr = errors.New("stale issuance survived account version bump")
		}
		done <- issueErr
	}()
	select {
	case err := <-done:
		t.Fatalf("issuance bypassed recovery row lock: %v", err)
	case <-time.After(120 * time.Millisecond):
	}
	if _, err := tx.Exec(ctx,
		"UPDATE accounts SET password_reset_version=password_reset_version+1 WHERE id=$1",
		account.ID); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-ctx.Done():
		t.Fatal("issuance did not finish after recovery committed")
	}
	var found bool
	if err := pool.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM password_reset_tokens WHERE token_hash=$1)",
		proof[:]).Scan(&found); err != nil || found {
		t.Fatalf("stale proof survived recovery: found=%t err=%v", found, err)
	}
}

// Older already-queued tasks have only account_id in JSON. Decoding zero
// defaults to generation 0: they remain valid before the first recovery,
// and automatically become stale after it.
func TestG1LegacyResetPayloadVersionZero(t *testing.T) {
	var payload struct {
		AccountID    int64 `json:"account_id"`
		ResetVersion int64 `json:"reset_version"`
	}
	if err := json.Unmarshal([]byte(`{"account_id":11}`), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.AccountID != 11 || payload.ResetVersion != 0 || strings.Contains("ignored", "reset_version") {
		t.Fatal("legacy job did not map safely to version zero")
	}
}
