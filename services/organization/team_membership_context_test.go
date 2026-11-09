package organization

import (
	"context"
	"errors"
	"testing"

	auth_model "github.com/datvtph41107/bdspro/models/auth"
	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/google/uuid"
)

func TestTeamMemberServiceRejectsMissingActorOrStore(t *testing.T) {
	if _, err := AddTeamMember(context.Background(), 7, 11); !errors.Is(err, auth_model.ErrSessionUnavailable) {
		t.Fatalf("add without actor returned %v", err)
	}
	if err := RemoveTeamMember(context.Background(), 7, 11); !errors.Is(err, auth_model.ErrSessionUnavailable) {
		t.Fatalf("remove without actor returned %v", err)
	}
	ctx := auth_model.WithActor(context.Background(), auth_model.Actor{AccountID: 42, SessionID: uuid.New()})
	if _, err := AddTeamMember(ctx, 7, 11); !errors.Is(err, db.ErrStoreMissing) {
		t.Fatalf("add without Store returned %v", err)
	}
	if err := RemoveTeamMember(ctx, 7, 11); !errors.Is(err, db.ErrStoreMissing) {
		t.Fatalf("remove without Store returned %v", err)
	}
}
