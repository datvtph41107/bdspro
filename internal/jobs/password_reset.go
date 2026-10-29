package jobs

import (
    "context"
    "crypto/rand"
    "crypto/sha256"
    "encoding/base64"
    "fmt"
    "html"
    "net/url"

    db "github.com/datvtph41107/bdspro/internal/database"
    "github.com/datvtph41107/bdspro/internal/mail"
)

// PasswordResetStore is the minimal data needed to fulfill a reset intent.
// The Asynq payload is an account id only, never the reset proof.
type PasswordResetStore interface {
    GetAccountByID(ctx context.Context, id int64) (db.Account, error)
    CreatePasswordResetToken(ctx context.Context, arg db.CreatePasswordResetTokenParams) error
}

// SendPasswordResetEmail issues the proof when executing a durable job.
// Provider failures return an error for native Asynq retry.
func SendPasswordResetEmail(ctx context.Context, store PasswordResetStore, sender mail.EmailSender, verifyURL string, accountID int64) error {
    account, err := store.GetAccountByID(ctx, accountID)
    if err != nil { return err }
    if !account.IsEmailVerified || account.Email == "" { return nil }
    raw := make([]byte, 32)
    if _, err := rand.Read(raw); err != nil { return err }
    secret := base64.RawURLEncoding.EncodeToString(raw)
    digest := sha256.Sum256([]byte(secret))
    link, err := url.Parse(verifyURL)
    if err != nil { return err }
    link.Path = "/users/password/reset"
    link.RawQuery = url.Values{"token": {secret}}.Encode()
    if err := store.CreatePasswordResetToken(ctx, db.CreatePasswordResetTokenParams{
        TokenHash: digest[:], AccountID: account.ID,
    }); err != nil { return err }
    content := fmt.Sprintf("Set or reset your BDSPro password within 15 minutes: <a href=%q>Set password</a>", html.EscapeString(link.String()))
    return sender.SendEmail("Reset your BDSPro password", content, []string{account.Email}, nil, nil, nil)
}
