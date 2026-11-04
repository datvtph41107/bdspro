package jobs

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"errors"
	"fmt"
	"html"
	"time"

	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/modules/secret"
	"github.com/datvtph41107/bdspro/services/mail"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// TeamInviteStore exposes only the query required to deliver an invitation.
// The proof never appears in the task payload or in PostgreSQL as plaintext.
type TeamInviteStore interface {
	GetTeamInviteForDelivery(context.Context, uuid.UUID) (db.GetTeamInviteForDeliveryRow, error)
}

func SendTeamInviteEmail(ctx context.Context, store TeamInviteStore, sender mail.EmailSender, signingKey string, id uuid.UUID) error {
	invite, err := store.GetTeamInviteForDelivery(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil // Expired, revoked, accepted or no longer eligible.
	}
	if err != nil {
		return err
	}
	token, err := secret.DeriveToken(signingKey, "team-invite:v1", id[:])
	if err != nil {
		return err
	}
	digest := sha256.Sum256([]byte(token))
	if !hmac.Equal(digest[:], invite.TokenHash) {
		return errors.New("invitation signing key does not match pending proof")
	}
	// No implicit public web route: the recipient must authenticate and call
	// the documented API with the received token after verifying their email.
	body := fmt.Sprintf(
		"You have been invited to the <b>%s</b> team in <b>%s</b>."+
			" Sign in with the verified email address and redeem this invitation"+
			" using POST /teams/invites/accept with token: <code>%s</code>."+
			" The invitation expires at %s.",
		html.EscapeString(invite.TeamName),
		html.EscapeString(invite.OrganizationName),
		html.EscapeString(token),
		html.EscapeString(invite.ExpiresAt.UTC().Format(time.RFC3339)),
	)
	return sender.SendEmail("BDSPro team invitation", body, []string{invite.InvitedEmail}, nil, nil, nil)
}
