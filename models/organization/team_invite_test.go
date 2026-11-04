package organization_test

import (
	"context"
	"errors"
	"testing"
	"time"

	organization "github.com/datvtph41107/bdspro/models/organization"
)

func TestTeamInvitationRejectsInvalidInput(t *testing.T) {
	_, err := organization.CreateTeamInvite(context.Background(), nil, 1, 2, "invalid-email", time.Hour)
	if !errors.Is(err, organization.ErrTeamInviteInvalid) {
		t.Fatalf("invalid invite email: %v", err)
	}
	_, err = organization.AcceptTeamInvite(context.Background(), nil, 1, "invalid token")
	if !errors.Is(err, organization.ErrTeamInviteInvalid) {
		t.Fatalf("invalid token accepted: %v", err)
	}
}
