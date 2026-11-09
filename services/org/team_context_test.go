package org

import (
	"context"
	"errors"
	"testing"

	auth_model "github.com/datvtph41107/bdspro/models/auth"
	db "github.com/datvtph41107/bdspro/models/db"
	org_model "github.com/datvtph41107/bdspro/models/organization"
	"github.com/google/uuid"
)

func TestTeamActionsUseDomainModelsAndRequestContext(t *testing.T) {
	if err := NewTeam(context.Background(), &org_model.Team{Name: " "}); !errors.Is(err, org_model.ErrTeamNameInvalid) {
		t.Fatalf("invalid create name used the store: %v", err)
	}
	if err := UpdateTeam(context.Background(), &org_model.Team{ID: 1, Name: " "}); !errors.Is(err, org_model.ErrTeamNameInvalid) {
		t.Fatalf("invalid update name used the store: %v", err)
	}
	if err := NewTeam(context.Background(), &org_model.Team{OrgID: 1, Name: "Valid"}); !errors.Is(err, auth_model.ErrSessionUnavailable) {
		t.Fatalf("missing actor authorized create: %v", err)
	}
	if err := UpdateTeam(context.Background(), &org_model.Team{ID: 1, Name: "Valid"}); !errors.Is(err, auth_model.ErrSessionUnavailable) {
		t.Fatalf("missing actor authorized update: %v", err)
	}
	ctx := auth_model.WithActor(context.Background(), auth_model.Actor{AccountID: 7, SessionID: uuid.New()})
	if err := NewTeam(ctx, &org_model.Team{OrgID: 1, Name: "Valid"}); !errors.Is(err, db.ErrStoreMissing) {
		t.Fatalf("missing request Store should fail before SQLC: %v", err)
	}
	if err := UpdateTeam(ctx, &org_model.Team{ID: 1, Name: "Valid"}); !errors.Is(err, db.ErrStoreMissing) {
		t.Fatalf("missing request Store should fail before SQLC: %v", err)
	}
}
