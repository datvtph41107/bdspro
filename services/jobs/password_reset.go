package jobs

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"html"
	"net/url"

	account_model "github.com/datvtph41107/bdspro/models/account"
	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/services/mail"
)

// SendPasswordResetEmail retains the pre-versioned entry point for callers
// operating on a never-recovered account (generation zero). Worker transport
// MUST use SendPasswordResetEmailForVersion with the persisted generation.
func SendPasswordResetEmail(ctx context.Context, sender mail.EmailSender, verifyURL string, accountID int64) error {
	return SendPasswordResetEmailForVersion(ctx, sender, verifyURL, accountID, 0)
}

// SendPasswordResetEmailForVersion only issues proofs for the recovery version
// captured when the durable job was requested. Provider failures remain retryable.
func SendPasswordResetEmailForVersion(ctx context.Context, sender mail.EmailSender, verifyURL string, accountID, resetVersion int64) error {
	account, err := account_model.ReadByID(ctx, accountID)
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
	q, err := db.GetQueries(ctx)
	if err != nil { return err }
	affected, err := q.IssuePasswordResetTokenIfCurrentVersion(ctx, db.IssuePasswordResetTokenIfCurrentVersionParams{
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
