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

func TestCreateTeamRejectsBadNameBeforeDatabase(t *testing.T) {
	_, err := organization.CreateTeam(context.Background(), nil, organization.CreateTeamOptions{Name: "   "})
	if !errors.Is(err, organization.ErrTeamNameInvalid) {
		t.Fatalf("invalid team name: %v", err)
	}
}

func TestListTeamsForDoerRejectsMissingActorOrStore(t *testing.T) {
	options := db.NewListOptions(1, 20, 20, 100)
	if _, err := organization.ListTeamsForDoer(context.Background(), 1, options); !errors.Is(err, auth_model.ErrSessionUnavailable) {
		t.Fatalf("missing actor error=%v", err)
	}
	ctx := auth_model.WithActor(context.Background(), auth_model.Actor{AccountID: 42, SessionID: uuid.New()})
	if _, err := organization.ListTeamsForDoer(ctx, 1, options); !errors.Is(err, db.ErrStoreMissing) {
		t.Fatalf("missing store error=%v", err)
	}
}
