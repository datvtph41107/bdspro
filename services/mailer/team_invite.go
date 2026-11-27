package mailer

import (
	"context"
	"strings"
	"time"

	"github.com/datvtph41107/bdspro/modules/mail"
)

func SendTeamInvitation(ctx context.Context, destination, proof string, expiresAt time.Time) error {
	if len(proof) != 43 || strings.ContainsAny(proof, "\r\n") {
		return ErrUnavailable
	}
	return mail.SendPlain(ctx, destination, "BDSPro team invitation",
		"Sign in to BDSPro with your invited account and enter this invitation token: "+proof+"\r\nExpires at "+expiresAt.UTC().Format(time.RFC3339)+".\r\nDo not share it.\r\n")
}
