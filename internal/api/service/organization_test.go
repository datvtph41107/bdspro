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

type organizationCreateStoreFake struct {
	db.Store
	queriedAccount int64
	createdWith db.CreateOrganizationTxParams
	createCount int
	accountErr error
}

func (s *organizationCreateStoreFake) GetAccountByID(_ context.Context, id int64) (db.Account, error) {
	s.queriedAccount = id
	if s.accountErr != nil {
		return db.Account{}, s.accountErr
	}
	return db.Account{ID: id, PersonID: 42}, nil
}

func (s *organizationCreateStoreFake) CreateOrganizationTx(_ context.Context, arg db.CreateOrganizationTxParams) (db.Organization, error) {
	s.createCount++
	s.createdWith = arg
	return db.Organization{PartyID: 100, Name: arg.Name}, nil
}

func TestOrganizationCreateUsesPersonAndTransaction(t *testing.T) {
	store := &organizationCreateStoreFake{}
	svc := NewOrganizationService(store)

	_, err := svc.Create(context.Background(), 11, "  ")
	if !errors.Is(err, apperrors.NameRequired) || store.createCount != 0 {
		t.Fatalf("blank organization name should not reach DB: %v", err)
	}

	created, err := svc.Create(context.Background(), 11, "  Example Company  ")
	if err != nil {
		t.Fatal(err)
	}
	if store.queriedAccount != 11 || store.createdWith.OwnerPersonID != 42 ||
		store.createdWith.Name != "Example Company" || store.createCount != 1 ||
		created.PartyID != 100 {
		t.Fatalf("incorrect create: account=%d args=%+v result=%+v", store.queriedAccount, store.createdWith, created)
	}

	store.accountErr = pgx.ErrNoRows
	_, err = svc.Create(context.Background(), 99, "Example Company")
	if !errors.Is(err, apperrors.AccountUnavailable) || store.createCount != 1 {
		t.Fatalf("missing account must return unauthorized without DB mutation: %v", err)
	}
}
