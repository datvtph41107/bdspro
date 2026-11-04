package organization_test

import (
	db "github.com/datvtph41107/bdspro/models/db"
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

func TestListTeamUsersRejectsUnboundedPaging(t *testing.T) {
	_, err := organization.ListTeamUsers(context.Background(), nil, 1, 2,
		db.ListOptions{Page: 1, PageSize: 20, ListAll: true})
	if !errors.Is(err, organization.ErrTeamPaginationInvalid) {
		t.Fatalf("ListAll must not reach the database: %v", err)
	}
}
