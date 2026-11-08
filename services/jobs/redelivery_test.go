package jobs

import (
	"context"
	"crypto/sha256"
	"errors"
	"testing"
	"time"

	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/modules/secret"
	"github.com/google/uuid"
)

// The SMTP server might accept a message while the worker observes a lost
// response. A second delivery is then mandatory under at-least-once semantics.
type ambiguousAcceptanceMailer struct {
	messages []string
}

func (m *ambiguousAcceptanceMailer) SendEmail(_ string, content string, _, _, _, _ []string) error {
	m.messages = append(m.messages, content)
	if len(m.messages) == 1 {
		return errors.New("simulated SMTP acknowledgement loss")
	}
	return nil
}

func TestVerificationRedeliveryKeepsTheSameProofAfterAmbiguousSMTP(t *testing.T) {
	store := &verificationStoreFake{
		account: db.Account{ID: 18, Email: "verify@example.test"},
		record: db.VerifyEmail{
			ID: 29, AccountID: 18, Email: "verify@example.test",
			SecretCode: "persisted-verification-proof",
			ExpiredAt: time.Now().Add(time.Hour),
		},
	}
	mailer := &ambiguousAcceptanceMailer{}
	ctx := context.Background()
	url := "https://example.test/users/verify_email"
	if err := SendVerificationEmail(ctx, store, mailer, url, 18); err == nil {
		t.Fatal("ambiguous SMTP acceptance should be retried")
	}
	if err := SendVerificationEmail(ctx, store, mailer, url, 18); err != nil {
		t.Fatal(err)
	}
	if len(mailer.messages) != 2 || mailer.messages[0] != mailer.messages[1] || store.tokens != 2 {
		t.Fatal("retry did not reuse the same persisted verification link")
	}
}

func TestInvitationRedeliveryKeepsTheSameProofAfterAmbiguousSMTP(t *testing.T) {
	id := uuid.New()
	key := "01234567890123456789012345678901"
	proof, err := secret.DeriveToken(key, "team-invite:v1", id[:])
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256([]byte(proof))
	store := &inviteDeliveryStore{row: db.GetTeamInviteForDeliveryRow{
		InvitedEmail: "invite@example.test",
		TokenHash: digest[:],
		ExpiresAt: time.Now().Add(time.Hour),
		TeamName: "Engineering",
		OrganizationName: "BDSPro",
	}}
	mailer := &ambiguousAcceptanceMailer{}
	ctx := context.Background()
	if err := SendTeamInviteEmail(ctx, store, mailer, key, id); err == nil {
		t.Fatal("ambiguous SMTP acceptance should be retried")
	}
	if err := SendTeamInviteEmail(ctx, store, mailer, key, id); err != nil {
		t.Fatal(err)
	}
	if len(mailer.messages) != 2 || mailer.messages[0] != mailer.messages[1] {
		t.Fatal("retry changed the invitation proof")
	}
}
