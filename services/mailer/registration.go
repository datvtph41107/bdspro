package mailer

import (
	"context"
	"strconv"

	token "github.com/datvtph41107/bdspro/modules/auth"
	"github.com/datvtph41107/bdspro/modules/mail"
)

func SendRegistrationCode(ctx context.Context, destination, code string, ttlMinutes int) error {
	if ttlMinutes <= 0 || !token.ValidRegistrationCode(code) {
		return ErrUnavailable
	}
	return mail.SendPlain(ctx, destination, "BDSPro email verification",
		"Enter this verification code in BDSPro: "+code+"\r\nIt expires in "+strconv.Itoa(ttlMinutes)+" minutes. If you did not request this, ignore this message.\r\n")
}
