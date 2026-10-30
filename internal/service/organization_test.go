package service

import (
	"context"
	"errors"
	"testing"

	db "github.com/datvtph41107/bdspro/internal/database"
	apperrors "github.com/datvtph41107/bdspro/internal/errors"
	"github.com/jackc/pgx/v5"
)

type organizationStoreFake struct {
	db.Store
	updateArg   db.UpdateOrganizationParams
	updateCalls int
	updateErr   error
	memberErr   error
}

func (s *organizationStoreFake) UpdateOrganization(_ context.Context, p db.UpdateOrganizationParams) (db.Organization, error) {
	s.updateCalls++
	s.updateArg = p
	if s.updateErr != nil {
		return db.Organization{}, s.updateErr
	}
	return db.Organization{PartyID: p.OrganizationID, Name: p.Name.String}, nil
}

func (s *organizationStoreFake) AddOrganizationMember(_ context.Context, _ db.AddOrganizationMemberParams) (db.OrganizationMember, error) {
	if s.memberErr != nil {
		return db.OrganizationMember{}, s.memberErr
	}
	return db.OrganizationMember{Role: "member"}, nil
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
	txCalls        int
	accountErr     error
	txErr          error
}

func (s *organizationCreateStoreFake) GetAccountByID(_ context.Context, id int64) (db.Account, error) {
	s.queriedAccount = id
	if s.accountErr != nil {
		return db.Account{}, s.accountErr
	}
	return db.Account{ID: id, PersonID: 42}, nil
}

func (s *organizationCreateStoreFake) ExecTx(_ context.Context, _ func(*db.Queries) error) error {
	s.txCalls++
	return s.txErr
}

// Pure preconditions are checked before a transaction is opened. The actual
// Party/Organization/owner sequence and rollback are exercised with PostgreSQL
// in organization_create_integration_test.go.
func TestOrganizationCreateValidatesBeforeTransaction(t *testing.T) {
	store := &organizationCreateStoreFake{}
	svc := NewOrganizationService(store)

	_, err := svc.Create(context.Background(), 11, "  ")
	if !errors.Is(err, apperrors.NameRequired) || store.txCalls != 0 {
		t.Fatalf("blank organization name opened transaction: %v", err)
	}

	store.accountErr = pgx.ErrNoRows
	_, err = svc.Create(context.Background(), 99, "Example Company")
	if !errors.Is(err, apperrors.AccountUnavailable) || store.txCalls != 0 {
		t.Fatalf("missing account opened transaction: %v", err)
	}

	store.accountErr = nil
	store.txErr = errors.New("transaction failed")
	_, err = svc.Create(context.Background(), 11, "  Example Company  ")
	if !errors.Is(err, store.txErr) || store.txCalls != 1 || store.queriedAccount != 11 {
		t.Fatalf("transaction error not propagated: calls=%d actor=%d err=%v", store.txCalls, store.queriedAccount, err)
	}
}
