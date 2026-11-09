package org

import (
	"context"
	"errors"
	"fmt"
	"testing"

	db "github.com/datvtph41107/bdspro/models/db"
	org_model "github.com/datvtph41107/bdspro/models/organization"
)

// A nested Organization action must reuse the parent's SQLC executor. Returning
// success after ignoring an inner error may never commit the outer transaction.
func TestOrganizationNestedTransactionIsRollbackOnlyAfterInnerError(t *testing.T) {
	ctx, pool, store := membershipTestDB(t)
	d := newOrganizationRaceCase(t, ctx, pool, store)
	actorCtx := organizationActorContext(ctx, store, d.accountID, d.sessionID)

	err := db.WithTx(actorCtx, func(txCtx context.Context) error {
		member, err := AddMember(txCtx, d.orgID, d.targetPersonID)
		if err != nil {
			return err
		}
		if member.PersonID != d.targetPersonID {
			return fmt.Errorf("wrong recipient person: %d", member.PersonID)
		}
		_, innerErr := AddMember(txCtx, d.orgID, d.targetPersonID)
		if !errors.Is(innerErr, org_model.ErrMembershipExists) {
			return fmt.Errorf("duplicate membership result: %w", innerErr)
		}
		// Deliberately ignore the duplicate error; db.WithTx must still
		// poison the parent transaction instead of committing the first write.
		return nil
	})
	if !errors.Is(err, org_model.ErrMembershipExists) {
		t.Fatalf("ignored nested error did not roll back parent: %v", err)
	}
	verifyOrganizationWrite(t, ctx, pool, d, "add_member", false)
}

// Nested success is provisional until the outermost transaction commits.
// This prevents an HTTP action from publishing an uncommitted Organization.
func TestOrganizationNestedCreationRollsBackWithParent(t *testing.T) {
	ctx, pool, store := membershipTestDB(t)
	d := newOrganizationRaceCase(t, ctx, pool, store)
	actorCtx := organizationActorContext(ctx, store, d.accountID, d.sessionID)
	forceRollback := errors.New("forced outer rollback")

	err := db.WithTx(actorCtx, func(txCtx context.Context) error {
		organization := &org_model.Organization{Name: d.createdName}
		if err := NewOrganization(txCtx, organization); err != nil {
			return err
		}
		if organization.ID == 0 || organization.Name != d.createdName {
			return fmt.Errorf("nested organization was not populated: %+v", organization)
		}
		return forceRollback
	})
	if !errors.Is(err, forceRollback) {
		t.Fatalf("outer transaction did not roll back: %v", err)
	}
	verifyOrganizationWrite(t, ctx, pool, d, "create", false)
}
