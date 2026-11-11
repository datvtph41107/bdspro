package jobs

import (
	"context"
	"errors"
	"fmt"
	"html"
	"net/url"

	account_model "github.com/datvtph41107/bdspro/models/account"
	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/modules/secret"
	"github.com/datvtph41107/bdspro/services/mail"
)

// SendVerificationEmail performs the business operation. Retry and transport
// policy belong to Asynq; this function can be exercised without Redis.
func SendVerificationEmail(
	ctx context.Context,
	mailer mail.EmailSender,
	verifyEmailURL string,
	accountID int64,
) error {
	account, err := account_model.ReadByID(ctx, accountID)
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
	record, err := account_model.GetOrCreateVerifyEmail(ctx, db.CreateVerifyEmailParams{
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
	content := fmt.Sprintf("Welcome to BDSPro. Please verify your email: <a href=%q>Verify email</a>", html.EscapeString(link.String()))
	return mailer.SendEmail("Verify your BDSPro email", content, []string{account.Email}, nil, nil, nil)
}
