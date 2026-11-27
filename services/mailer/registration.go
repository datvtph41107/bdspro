package mailer

import (
	"context"
	"strings"

	"github.com/datvtph41107/bdspro/modules/mail"
)

func SendRegistrationCode(ctx context.Context, destination, code string) error {
	if len(code) != 8 || strings.IndexFunc(code, func(r rune) bool { return r < '0' || r > '9' }) >= 0 {
		return ErrUnavailable
	}
	return mail.SendPlain(ctx, destination, "BDSPro email verification",
		"Enter this verification code in BDSPro: "+code+"\r\nIt expires in 5 minutes. If you did not request this, ignore this message.\r\n")
}
