package organization

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	org_model "github.com/datvtph41107/bdspro/models/organization"
)

func TestMalformedInviteProofNeverTouchesStore(t *testing.T) {
	service := NewTeamService(nil)
	if _, err := service.AcceptTeamInvite(context.Background(), 1,
		uuid.Nil, "this is not hex"); !errors.Is(err, org_model.ErrTeamInviteInvalid) {
		t.Fatalf("malformed proof bypassed preflight validation: %v", err)
	}
}
