package organization_test

import (
	"context"
	"errors"
	"testing"

	organization "github.com/datvtph41107/bdspro/models/organization"
)

func TestCreateTeamRejectsBadNameBeforeDatabase(t *testing.T) {
	_, err := organization.CreateTeam(context.Background(), nil, organization.CreateTeamOptions{Name: "   "})
	if !errors.Is(err, organization.ErrTeamNameInvalid) {
		t.Fatalf("invalid team name: %v", err)
	}
}
