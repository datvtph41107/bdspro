package organization

import (
	"context"
	"errors"
	"testing"

	auth_model "github.com/datvtph41107/bdspro/models/auth"
	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/google/uuid"
)

func TestContextScopedOrganizationReadsRequireVerifiedActor(t *testing.T) {
	ctx := context.Background()
	if _, err := ListForDoer(ctx); !errors.Is(err, auth_model.ErrSessionUnavailable) {
		t.Fatalf("list without actor: %v", err)
	}
	if _, err := GetForDoer(ctx, 1); !errors.Is(err, auth_model.ErrSessionUnavailable) {
		t.Fatalf("organization read without actor: %v", err)
	}
	if _, err := GetTeamForDoer(ctx, 1); !errors.Is(err, auth_model.ErrSessionUnavailable) {
		t.Fatalf("team read without actor: %v", err)
	}
}

func TestContextScopedOrganizationReadsRequireInstalledStore(t *testing.T) {
	ctx := auth_model.WithActor(context.Background(), auth_model.Actor{
		AccountID: 1, SessionID: uuid.New(),
	})
	if _, err := ListForDoer(ctx); !errors.Is(err, db.ErrStoreMissing) {
		t.Fatalf("list without Store: %v", err)
	}
	if _, err := GetForDoer(ctx, 1); !errors.Is(err, db.ErrStoreMissing) {
		t.Fatalf("organization read without Store: %v", err)
	}
	if _, err := GetTeamForDoer(ctx, 1); !errors.Is(err, db.ErrStoreMissing) {
		t.Fatalf("team read without Store: %v", err)
	}
}
