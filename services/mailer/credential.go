package mailer

import (
	"context"

	"github.com/datvtph41107/bdspro/modules/mail"
)

func SendCredentialNotice(ctx context.Context, destination string) error {
	return mail.SendPlain(ctx, destination, "BDSPro account security change",
		"A sign-in method on your BDSPro account changed. If you did not make this change, contact support.\r\n")
}
