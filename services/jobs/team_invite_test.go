package jobs

import (
	"context"
	"crypto/sha256"
	"strings"
	"testing"
	"time"

	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/modules/secret"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type inviteDeliveryStore struct {
	row db.GetTeamInviteForDeliveryRow
	err error
}
func (s *inviteDeliveryStore) GetTeamInviteForDelivery(_ context.Context, _ uuid.UUID) (db.GetTeamInviteForDeliveryRow, error) {
	return s.row, s.err
}

type inviteMailer struct {
	to []string
	body string
	calls int
}
func (m *inviteMailer) SendEmail(subject, content string, to, cc, bcc, attachFiles []string) error {
	m.calls++
	m.to = to
	m.body = content
	return nil
}

func TestTeamInviteJobRegeneratesProofWithoutPersistingPlaintext(t *testing.T) {
	id := uuid.New()
	key := "01234567890123456789012345678901"
	token, err := secret.DeriveToken(key, "team-invite:v1", id[:])
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256([]byte(token))
	store := &inviteDeliveryStore{row: db.GetTeamInviteForDeliveryRow{
		InvitedEmail: "member@example.test",
		TokenHash: hash[:],
		ExpiresAt: time.Now().Add(time.Hour),
		TeamName: "<script>Engineering</script>",
		OrganizationName: "Example",
	}}
	sender := &inviteMailer{}
	if err := SendTeamInviteEmail(context.Background(), store, sender, key, id); err != nil {
		t.Fatal(err)
	}
	if sender.calls != 1 || len(sender.to) != 1 || sender.to[0] != "member@example.test" {
		t.Fatalf("wrong invitation recipient: %+v", sender)
	}
	if !strings.Contains(sender.body, token) {
		t.Fatal("worker did not reconstruct the usable invitation proof")
	}
	if strings.Contains(sender.body, "<script>") {
		t.Fatal("untrusted team name was not HTML-escaped")
	}
}

func TestTeamInviteJobSkipsExpiredAndDetectsKeyMismatch(t *testing.T) {
	id := uuid.New()
	sender := &inviteMailer{}
	store := &inviteDeliveryStore{err: pgx.ErrNoRows}
	if err := SendTeamInviteEmail(context.Background(), store, sender, "01234567890123456789012345678901", id); err != nil {
		t.Fatal(err)
	}
	if sender.calls != 0 {
		t.Fatal("expired invite generated an email")
	}
	store.err = nil
	store.row = db.GetTeamInviteForDeliveryRow{TokenHash: make([]byte, 32)}
	if err := SendTeamInviteEmail(context.Background(), store, sender, "01234567890123456789012345678901", id); err == nil {
		t.Fatal("rotated signing key silently generated an unusable invitation")
	}
	if sender.calls != 0 {
		t.Fatal("invalid proof was sent")
	}
}
