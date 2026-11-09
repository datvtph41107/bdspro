package organization_test

import (
	"context"
	"errors"
	"testing"

	auth_model "github.com/datvtph41107/bdspro/models/auth"
	db "github.com/datvtph41107/bdspro/models/db"
	organization "github.com/datvtph41107/bdspro/models/organization"
	"github.com/google/uuid"
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

func TestListTeamUsersForDoerRequiresActorAndStore(t *testing.T) {
	opts := db.NewListOptions(1, 20, 20, 100)
	if _, err := organization.ListTeamUsersForDoer(context.Background(), 1, opts); !errors.Is(err, auth_model.ErrSessionUnavailable) {
		t.Fatalf("missing actor: %v", err)
	}
	ctx := auth_model.WithActor(context.Background(), auth_model.Actor{AccountID: 42, SessionID: uuid.New()})
	if _, err := organization.ListTeamUsersForDoer(ctx, 1, opts); !errors.Is(err, db.ErrStoreMissing) {
		t.Fatalf("missing Store: %v", err)
	}
}
