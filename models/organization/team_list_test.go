package organization_test

import (
	"context"
	"errors"
	"testing"

	db "github.com/datvtph41107/bdspro/models/db"
	organization "github.com/datvtph41107/bdspro/models/organization"
)

func TestListTeamsRejectsUnboundedOptions(t *testing.T) {
	_, err := organization.ListTeams(context.Background(), nil, 1, 2,
		db.ListOptions{Page: 1, PageSize: 20, ListAll: true})
	if !errors.Is(err, organization.ErrTeamPaginationInvalid) {
		t.Fatalf("ListAll should be rejected: %v", err)
	}
}
