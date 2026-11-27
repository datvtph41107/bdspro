package mailer

import (
	"context"
	"strconv"

	token "github.com/datvtph41107/bdspro/modules/auth"
	"github.com/datvtph41107/bdspro/modules/mail"
)

func SendPasswordReset(ctx context.Context, destination, proof string, ttlMinutes int) error {
	if ttlMinutes <= 0 || !token.ValidOpaqueProof(proof) {
		return ErrUnavailable
	}
	return mail.SendPlain(ctx, destination, "BDSPro password recovery",
		"Enter this recovery token in BDSPro: "+proof+"\r\nIt expires in "+strconv.Itoa(ttlMinutes)+" minutes. If you did not request this, ignore this message.\r\n")
}

func SendPasswordSetProof(ctx context.Context, destination, proof string, ttlMinutes int) error {
	if ttlMinutes <= 0 || !token.ValidOpaqueProof(proof) {
		return ErrUnavailable
	}
	return mail.SendPlain(ctx, destination, "BDSPro password setup",
		"Enter this proof only in your signed-in BDSPro password setup: "+proof+"\r\nIt expires in "+strconv.Itoa(ttlMinutes)+" minutes. Do not share it.\r\n")
}
