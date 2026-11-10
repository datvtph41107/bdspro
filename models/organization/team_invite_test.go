package organization_test

import (
	"context"
	"encoding/json"
	"strings"
	"errors"
	"testing"
	"time"

	organization "github.com/datvtph41107/bdspro/models/organization"
	"github.com/google/uuid"
)

func TestTeamInvitationRejectsInvalidInput(t *testing.T) {
	_, err := organization.CreateTeamInvite(context.Background(), nil, 1, 2, "invalid-email", time.Hour, "01234567890123456789012345678901")
	if !errors.Is(err, organization.ErrTeamInviteInvalid) {
		t.Fatalf("invalid invite email: %v", err)
	}
	_, err = organization.AcceptTeamInvite(context.Background(), nil, 1, "invalid token")
	if !errors.Is(err, organization.ErrTeamInviteInvalid) {
		t.Fatalf("invalid token accepted: %v", err)
	}
}

func TestCreatedTeamInviteNeverSerializesProof(t *testing.T) {
	value := organization.CreatedTeamInvite{
		Invite: organization.TeamInvite{
			ID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			TeamID: 8, InvitedEmail: "member@example.test",
		},
		Token: "NEVER_LEAK_ME",
	}
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "NEVER_LEAK_ME") || strings.Contains(string(raw), "TokenHash") {
		t.Fatalf("invitation proof leaked into JSON: %s", raw)
	}
	if value.Token == "" || value.Invite.TeamID != 8 {
		t.Fatal("delivery metadata lost")
	}
}
