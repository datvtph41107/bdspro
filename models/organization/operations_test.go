package organization

import (
	"context"
	"errors"
	"testing"

	db "github.com/datvtph41107/bdspro/models/db"
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
	ctx := context.Background()

	if _, err := UpdateOrganization(ctx, store, 11, 23, "  "); !errors.Is(err, ErrNameRequired) {
		t.Fatalf("blank name: %v", err)
	}
	if store.updateCalls != 0 {
		t.Fatal("invalid request reached database")
	}
	updated, err := UpdateOrganization(ctx, store, 11, 23, "  New name  ")
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != "New name" || store.updateArg.ActorAccountID != 11 ||
		store.updateArg.OrganizationID != 23 || store.updateArg.Name.String != "New name" {
		t.Fatalf("actor/name mismatch: %+v %+v", updated, store.updateArg)
	}

	store.updateErr = pgx.ErrNoRows
	if _, err := UpdateOrganization(ctx, store, 11, 23, "Denied"); !errors.Is(err, ErrOrganizationOwnerRequired) {
		t.Fatalf("SQL owner denial must map to forbidden: %v", err)
	}
}

func TestOrganizationMemberDeniedBySQL(t *testing.T) {
	store := &organizationStoreFake{memberErr: pgx.ErrNoRows}
	if _, err := AddOrgUser(context.Background(), store, 11, 23, 31); !errors.Is(err, ErrOrganizationOwnerRequired) {
		t.Fatalf("non-owner must be denied: %v", err)
	}
}

type organizationCreateStoreFake struct {
	db.Store
	txCalls int
	txErr   error
}

func (s *organizationCreateStoreFake) ExecTx(_ context.Context, _ func(*db.Queries) error) error {
	s.txCalls++
	return s.txErr
}

// Invalid names are rejected before opening a transaction. Creator-account
// existence is deliberately checked INSIDE the transaction; PostgreSQL tests
// cover the missing-account and concurrent-deletion cases.
func TestOrganizationCreateValidatesBeforeTransaction(t *testing.T) {
	store := &organizationCreateStoreFake{}

	_, err := CreateOrganization(context.Background(), store, 11, "  ")
	if !errors.Is(err, ErrNameRequired) || store.txCalls != 0 {
		t.Fatalf("blank organization name opened transaction: %v", err)
	}

	store.txErr = errors.New("transaction failed")
	created, err := CreateOrganization(context.Background(), store, 11, "  Example Company  ")
	if !errors.Is(err, store.txErr) || store.txCalls != 1 || created.PartyID != 0 {
		t.Fatalf("transaction error not propagated or result leaked: calls=%d created=%+v err=%v", store.txCalls, created, err)
	}
}
