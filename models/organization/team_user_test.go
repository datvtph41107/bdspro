package organization_test

import (
	"context"
	"errors"
	"testing"

	organization "github.com/datvtph41107/bdspro/models/organization"
)

func TestTeamMemberErrorsAreOwnedByOrganization(t *testing.T) {
	for _, err := range []error{organization.ErrTeamMemberExists, organization.ErrTeamMemberNotFound} {
		if !errors.Is(errors.Join(err, context.Canceled), err) {
			t.Fatalf("wrapped team member error identity lost: %v", err)
		}
	}
}
