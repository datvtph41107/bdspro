package jobs

import (
	"context"
	"crypto/sha256"
	"errors"
	"regexp"
	"testing"

	account_model "github.com/datvtph41107/bdspro/models/account"
	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/google/uuid"
)

func passwordResetTestAccount(t *testing.T, verified bool) (context.Context, int64) {
	t.Helper()
	ctx, _, q := g1ResetStore(t)
	account, err := account_model.Register(ctx, account_model.RegisterParams{
		Email: uuid.NewString() + "@example.test", PasswordHash: "test-password-hash",
	})
	if err != nil {
		t.Fatal(err)
	}
	if verified {
		if _, err := q.MarkEmailVerified(ctx, db.MarkEmailVerifiedParams{
			ID: account.ID, Email: account.Email,
		}); err != nil {
			t.Fatal(err)
		}
	}
	return ctx, account.ID
}

func TestPasswordResetJobGeneratesProofInWorkerAndRetriesSMTP(t *testing.T) {
	ctx, accountID := passwordResetTestAccount(t, true)
	sender := &fakeMailSender{fail: errors.New("SMTP unavailable")}
	url := "https://example.test/users/verify_email"
	if err := SendPasswordResetEmailForVersion(ctx, sender, url, accountID, 0); err == nil {
		t.Fatal("SMTP failure must be retryable")
	}
	if len(sender.content) != 0 {
		t.Fatal("failed SMTP unexpectedly recorded successful delivery")
	}
	sender.fail = nil
	if err := SendPasswordResetEmailForVersion(ctx, sender, url, accountID, 0); err != nil {
		t.Fatal(err)
	}
	if len(sender.content) != 1 {
		t.Fatal("retry failed to send usable proof")
	}
	match := regexp.MustCompile(`token=([A-Za-z0-9_-]{43})`).FindStringSubmatch(sender.content[0])
	if len(match) != 2 {
		t.Fatal("email missing one-time link")
	}
	digest := sha256.Sum256([]byte(match[1]))
	_, pool, _ := g1ResetStore(t)
	var found bool
	if err := pool.QueryRow(ctx,
		"SELECT EXISTS (SELECT 1 FROM password_reset_tokens WHERE account_id=$1 AND token_hash=$2)",
		accountID, digest[:],
	).Scan(&found); err != nil || !found {
		t.Fatalf("reset proof not persisted: found=%t err=%v", found, err)
	}
}

func TestPasswordResetJobSkipsUnverifiedAccounts(t *testing.T) {
	ctx, accountID := passwordResetTestAccount(t, false)
	sender := &fakeMailSender{}
	if err := SendPasswordResetEmailForVersion(ctx, sender,
		"https://example.test/users/verify_email", accountID, 0); err != nil {
		t.Fatal(err)
	}
	if len(sender.content) != 0 {
		t.Fatal("unverified email received password reset proof")
	}
}

func TestPasswordResetJobPropagatesDatabaseFailure(t *testing.T) {
	ctx, accountID := passwordResetTestAccount(t, true)
	sender := &fakeMailSender{}
	abort := errors.New("rollback database failure test")
	err := db.WithTx(ctx, func(txCtx context.Context) error {
		executor, err := db.GetEngine(txCtx)
		if err != nil {
			return err
		}
		// PostgreSQL marks the transaction aborted after a failed statement.
		if _, err := executor.Exec(txCtx, "SELECT 1/0"); err == nil {
			t.Fatal("expected PostgreSQL statement failure")
		}
		if err := SendPasswordResetEmailForVersion(txCtx, sender,
			"https://example.test/users/verify_email", accountID, 0); err == nil {
			t.Fatal("failed PostgreSQL transaction did not propagate an error")
		}
		return abort
	})
	if !errors.Is(err, abort) {
		t.Fatalf("transaction rollback error: %v", err)
	}
	if len(sender.content) != 0 {
		t.Fatal("database failure allowed SMTP delivery")
	}
}
