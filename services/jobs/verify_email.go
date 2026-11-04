package jobs

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/url"

	account_model "github.com/datvtph41107/bdspro/models/account"
	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/modules/secret"
	"github.com/datvtph41107/bdspro/services/mail"
	"github.com/datvtph41107/bdspro/templates"
)

// VerificationStore is the data contract required by email verification.
// A job does not depend on the application's complete database Store.
type VerificationStore interface {
	account_model.Transactioner
	GetAccountByID(ctx context.Context, id int64) (db.Account, error)
}

// SendVerificationEmail performs the business operation. Retry and transport
// policy belong to Asynq; this function can be exercised without Redis.
func SendVerificationEmail(
	ctx context.Context,
	store VerificationStore,
	mailer mail.EmailSender,
	verifyEmailURL string,
	accountID int64,
) error {
	account, err := store.GetAccountByID(ctx, accountID)
	if err != nil {
		return err
	}
	if account.IsEmailVerified {
		return nil
	}

	secret, err := secret.RandomHex(32)
	if err != nil {
		return err
	}
	record, err := account_model.GetOrCreateVerifyEmail(ctx, store, db.CreateVerifyEmailParams{
		AccountID: account.ID, Email: account.Email, SecretCode: secret,
	})
	if errors.Is(err, account_model.ErrEmailAlreadyVerified) {
		return nil
	}
	if err != nil {
		return err
	}

	link, err := url.Parse(verifyEmailURL)
	if err != nil {
		return fmt.Errorf("invalid verification URL: %w", err)
	}
	query := link.Query()
	query.Set("email_id", fmt.Sprint(record.ID))
	query.Set("secret_code", record.SecretCode)
	link.RawQuery = query.Encode()
	var body bytes.Buffer
	if err := templates.VerifyEmailMail.Execute(&body, struct {
		VerificationURL string
	}{VerificationURL: link.String()}); err != nil {
		return fmt.Errorf("render verification email: %w", err)
	}
	return mailer.SendEmail("Verify your BDSPro email", body.String(), []string{account.Email}, nil, nil, nil)
}
