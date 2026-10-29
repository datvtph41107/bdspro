package jobs

import (
    "context"
    "crypto/sha256"
    "errors"
    "regexp"
    "testing"
    "time"

    db "github.com/datvtph41107/bdspro/internal/database"
)

type passwordResetStoreFake struct {
    account db.Account
    tokenHashes [][]byte
}
func (f *passwordResetStoreFake) GetAccountByID(_ context.Context, _ int64) (db.Account, error) {
    return f.account, nil
}
func (f *passwordResetStoreFake) CreatePasswordResetToken(_ context.Context, arg db.CreatePasswordResetTokenParams) error {
    f.tokenHashes = append(f.tokenHashes, append([]byte(nil), arg.TokenHash...))
    return nil
}

type passwordResetSenderFake struct {
    fail bool
    content string
}
func (f *passwordResetSenderFake) SendEmail(_ string, content string, _, _, _, _ []string) error {
    if f.fail { return errors.New("SMTP unavailable") }
    f.content = content
    return nil
}

func TestPasswordResetWorkerRetriesWithoutPersistingPlaintextToken(t *testing.T) {
    store := &passwordResetStoreFake{account: db.Account{ID: 42, Email: "verified@example.test", IsEmailVerified: true, CreatedAt: time.Now()}}
    sender := &passwordResetSenderFake{fail: true}
    if err := SendPasswordReset(context.Background(), store, sender, "http://localhost:8080/users/verify_email", 42); err == nil {
        t.Fatal("transient SMTP failure was suppressed instead of retried")
    }
    sender.fail = false
    if err := SendPasswordReset(context.Background(), store, sender, "http://localhost:8080/users/verify_email", 42); err != nil {
        t.Fatal(err)
    }
    if len(store.tokenHashes) != 2 { t.Fatalf("expected distinct delivery attempt proofs, got %d", len(store.tokenHashes)) }
    m := regexp.MustCompile(`token=([A-Za-z0-9_-]{43})`).FindStringSubmatch(sender.content)
    if len(m) != 2 { t.Fatal("email has no reset token") }
    hash := sha256.Sum256([]byte(m[1]))
    if string(store.tokenHashes[1]) != string(hash[:]) { t.Fatal("database token hash does not verify delivered token") }
    if string(store.tokenHashes[1]) == m[1] { t.Fatal("plaintext token persisted") }
}
