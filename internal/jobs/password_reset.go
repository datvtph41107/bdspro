package jobs

import (
    "context"
    "crypto/rand"
    "crypto/sha256"
    "encoding/base64"
    "errors"
    "fmt"
    "html"
    "net/url"

    db "github.com/datvtph41107/bdspro/internal/database"
    "github.com/datvtph41107/bdspro/internal/mail"
    "github.com/jackc/pgx/v5"
)

type PasswordResetStore interface {
    GetAccountByID(ctx context.Context, id int64) (db.Account, error)
    CreatePasswordResetToken(ctx context.Context, arg db.CreatePasswordResetTokenParams) error
}

// SendPasswordReset generates a secret only inside the worker. The durable
// outbox payload contains AccountID only, never bearer secrets or passwords.
// At-least-once retry can issue multiple links; ResetPasswordTx consumes one
// and revokes every remaining reset token for the same account atomically.
func SendPasswordReset(ctx context.Context, store PasswordResetStore, sender mail.EmailSender, verifyURL string, accountID int64) error {
    account, err := store.GetAccountByID(ctx, accountID)
    if errors.Is(err, pgx.ErrNoRows) { return nil }
    if err != nil { return err }
    if !account.IsEmailVerified || account.Email == "" { return nil }

    raw := make([]byte, 32)
    if _, err := rand.Read(raw); err != nil { return err }
    token := base64.RawURLEncoding.EncodeToString(raw)
    digest := sha256.Sum256([]byte(token))
    if err := store.CreatePasswordResetToken(ctx, db.CreatePasswordResetTokenParams{
        TokenHash: digest[:], AccountID: account.ID,
    }); err != nil { return err }

    link, err := url.Parse(verifyURL)
    if err != nil { return fmt.Errorf("invalid password reset URL: %w", err) }
    link.Path = "/users/password/reset"
    link.RawQuery = url.Values{"token": {token}}.Encode()
    content := fmt.Sprintf(
        "Set or reset your BDSPro password within 15 minutes: <a href=%q>Set password</a>",
        html.EscapeString(link.String()),
    )
    return sender.SendEmail("Reset your BDSPro password", content, []string{account.Email}, nil, nil, nil)
}
