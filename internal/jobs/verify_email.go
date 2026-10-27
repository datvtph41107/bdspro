package jobs

import (
	"context"
	"errors"
	"fmt"
	"html"
	"net/url"

	db "github.com/datvtph41107/bdspro/internal/db/sqlc"
	"github.com/datvtph41107/bdspro/internal/mail"
	"github.com/datvtph41107/bdspro/internal/util"
)

// SendVerificationEmail is the job action. Transport scheduling and retry
// decisions remain in worker, while the job can reuse existing DB/mail behavior.
func SendVerificationEmail(
	ctx context.Context,
	store db.Store,
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
