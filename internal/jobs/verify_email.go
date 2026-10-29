package jobs

import (
	"context"
	"errors"
	"fmt"
	"html"
	"net/url"

	db "github.com/datvtph41107/bdspro/internal/database"
	"github.com/datvtph41107/bdspro/internal/mail"
	"github.com/datvtph41107/bdspro/internal/util"
)

// VerificationStore is the data contract required by email verification.
// A job does not depend on the application's complete database Store.
type VerificationStore interface {
	GetAccountByID(ctx context.Context, id int64) (db.GetAccountByIDRow, error)
	GetOrCreateVerifyEmailTx(ctx context.Context, arg db.CreateVerifyEmailParams) (db.VerifyEmail, error)
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

	secret, err := util.RandomSecret()
	if err != nil {
		return err
	}
	record, err := store.GetOrCreateVerifyEmailTx(ctx, db.CreateVerifyEmailParams{
		AccountID: account.ID, Email: account.Email, SecretCode: secret,
	})
	if errors.Is(err, db.ErrEmailAlreadyVerified) {
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
