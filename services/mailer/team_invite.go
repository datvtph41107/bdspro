package mailer

import (
	"context"
	"time"

	token "github.com/datvtph41107/bdspro/modules/auth"
	"github.com/datvtph41107/bdspro/modules/mail"
)

func SendTeamInvitation(ctx context.Context, destination, proof string, expiresAt time.Time) error {
	if !token.ValidOpaqueProof(proof) {
		return ErrUnavailable
	}
	return mail.SendPlain(ctx, destination, "BDSPro team invitation",
		"Sign in to BDSPro with your invited account and enter this invitation token: "+proof+"\r\nExpires at "+expiresAt.UTC().Format(time.RFC3339)+".\r\nDo not share it.\r\n")
}
