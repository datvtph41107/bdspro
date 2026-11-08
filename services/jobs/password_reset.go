package jobs

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"html"
	"net/url"

	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/services/mail"
)

// PasswordResetStore is the minimal data needed to fulfill a reset intent.
// The Asynq payload is an account id only, never the reset proof.
type PasswordResetStore interface {
	GetAccountByID(ctx context.Context, id int64) (db.Account, error)
	IssuePasswordResetTokenIfCurrentVersion(ctx context.Context, arg db.IssuePasswordResetTokenIfCurrentVersionParams) (int64, error)
}

// SendPasswordResetEmail issues a proof only for the recovery version captured
// when the durable job was requested. Provider failures remain retryable.
func SendPasswordResetEmail(ctx context.Context, store PasswordResetStore, sender mail.EmailSender, verifyURL string, accountID, resetVersion int64) error {
	account, err := store.GetAccountByID(ctx, accountID)
	if err != nil {
		return err
	}
	if !account.IsEmailVerified || account.Email == "" {
		return nil
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return err
	}
	secret := base64.RawURLEncoding.EncodeToString(raw)
	digest := sha256.Sum256([]byte(secret))
	link, err := url.Parse(verifyURL)
	if err != nil {
		return err
	}
	link.Path = "/users/password/reset"
	link.RawQuery = url.Values{"token": {secret}}.Encode()
	affected, err := store.IssuePasswordResetTokenIfCurrentVersion(ctx, db.IssuePasswordResetTokenIfCurrentVersionParams{
		TokenHash: digest[:], AccountID: account.ID, ResetVersion: resetVersion,
	})
	if err != nil {
		return err
	}
	if affected != 1 {
		return nil // Revoked generation, missing account or unverified email.
	}
	content := fmt.Sprintf("Set or reset your BDSPro password within 15 minutes: <a href=%q>Set password</a>", html.EscapeString(link.String()))
	return sender.SendEmail("Reset your BDSPro password", content, []string{account.Email}, nil, nil, nil)
}
