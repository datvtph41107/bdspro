package service

import (
    "context"
    "errors"
    "testing"

    apperrors "github.com/datvtph41107/bdspro/internal/core/errors"
    db "github.com/datvtph41107/bdspro/internal/db/sqlc"
    "github.com/jackc/pgx/v5"
)

type organizationStoreFake struct {
    db.Store
    updateArg db.UpdateOrganizationParams
    updateCalls int
    updateErr error
    memberErr error
}

func (s *organizationStoreFake) UpdateOrganization(_ context.Context, p db.UpdateOrganizationParams) (db.Organization, error) {
    s.updateCalls++
    s.updateArg = p
    if s.updateErr != nil {
        return db.Organization{}, s.updateErr
    }
    return db.Organization{PartyID: p.OrganizationID, Name: p.Name.String}, nil
}

func (s *organizationStoreFake) AddOrganizationMember(_ context.Context, _ db.AddOrganizationMemberParams) (db.OrganizationMembership, error) {
    if s.memberErr != nil {
        return db.OrganizationMembership{}, s.memberErr
    }
    return db.OrganizationMembership{Role: "member"}, nil
}

func TestOrganizationUpdateValidatesAndCarriesActor(t *testing.T) {
    store := &organizationStoreFake{}
    service := NewOrganizationService(store)
    ctx := context.Background()

    if _, err := service.Update(ctx, 11, 23, "  "); !errors.Is(err, apperrors.NameRequired) {
        t.Fatalf("blank name: %v", err)
    }
    if store.updateCalls != 0 {
        t.Fatal("invalid request reached database")
    }
    updated, err := service.Update(ctx, 11, 23, "  New name  ")
    if err != nil {
        t.Fatal(err)
    }
    if updated.Name != "New name" || store.updateArg.ActorAccountID != 11 ||
        store.updateArg.OrganizationID != 23 || store.updateArg.Name.String != "New name" {
        t.Fatalf("actor/name mismatch: %+v %+v", updated, store.updateArg)
    }

    store.updateErr = pgx.ErrNoRows
    if _, err := service.Update(ctx, 11, 23, "Denied"); !errors.Is(err, apperrors.OrganizationOwnerRequired) {
        t.Fatalf("SQL owner denial must map to forbidden: %v", err)
    }
}

func TestOrganizationMemberDeniedBySQL(t *testing.T) {
    store := &organizationStoreFake{memberErr: pgx.ErrNoRows}
    service := NewOrganizationService(store)
    if _, err := service.AddMember(context.Background(), 11, 23, 31); !errors.Is(err, apperrors.OrganizationOwnerRequired) {
        t.Fatalf("non-owner must be denied: %v", err)
    }
}
