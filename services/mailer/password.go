package mailer

import (
	"context"
	"strings"

	"github.com/datvtph41107/bdspro/modules/mail"
)

func SendPasswordReset(ctx context.Context, destination, proof string) error {
	if len(proof) != 43 || strings.ContainsAny(proof, "\r\n") {
		return ErrUnavailable
	}
	return mail.SendPlain(ctx, destination, "BDSPro password recovery",
		"Enter this recovery token in BDSPro: "+proof+"\r\nIt expires in 15 minutes. If you did not request this, ignore this message.\r\n")
}

func SendPasswordSetProof(ctx context.Context, destination, proof string) error {
	if len(proof) != 43 || strings.ContainsAny(proof, "\r\n") {
		return ErrUnavailable
	}
	return mail.SendPlain(ctx, destination, "BDSPro password setup",
		"Enter this proof only in your signed-in BDSPro password setup: "+proof+"\r\nIt expires in 10 minutes. Do not share it.\r\n")
}
