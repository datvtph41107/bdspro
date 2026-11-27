package mailer

import (
	"context"
	"strconv"
	"strings"

	"github.com/datvtph41107/bdspro/modules/mail"
)

func SendRegistrationCode(ctx context.Context, destination, code string, ttlMinutes int) error {
	if ttlMinutes <= 0 || len(code) != 8 || strings.IndexFunc(code, func(r rune) bool { return r < '0' || r > '9' }) >= 0 {
		return ErrUnavailable
	}
	return mail.SendPlain(ctx, destination, "BDSPro email verification",
		"Enter this verification code in BDSPro: "+code+"\r\nIt expires in "+strconv.Itoa(ttlMinutes)+" minutes. If you did not request this, ignore this message.\r\n")
}
