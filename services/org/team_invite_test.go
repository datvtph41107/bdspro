package org

import (
	"context"
	"errors"
	"testing"

	org_model "github.com/datvtph41107/bdspro/models/organization"
)

func TestMalformedInviteProofNeverTouchesStore(t *testing.T) {
	if _, err := AcceptTeamInvite(context.Background(), "this is not hex"); !errors.Is(err, org_model.ErrTeamInviteInvalid) {
		t.Fatalf("malformed proof bypassed preflight validation: %v", err)
	}
}
