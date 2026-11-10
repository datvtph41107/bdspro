package organization

import (
	"context"
	"errors"
	"testing"

	account_model "github.com/datvtph41107/bdspro/models/account"
	auth_model "github.com/datvtph41107/bdspro/models/auth"
	db "github.com/datvtph41107/bdspro/models/db"
)

func TestListForDoerRequiresActingPrincipal(t *testing.T) {
	_, err := ListForDoer(context.Background())
	if !errors.Is(err, auth_model.ErrSessionUnavailable) {
		t.Fatalf("expected unverified principal error, got %v", err)
	}
}

func TestLockedCreatorCannotAuthorizeAnotherTransactionExecutor(t *testing.T) {
	lockedExecutor := &db.Queries{}
	differentExecutor := &db.Queries{}
	proof := LockedCreator{personID: 17, executor: lockedExecutor}

	// A proof acquired under the first executor must not authorize writes
	// through another executor; no SQL operation should be attempted.
	created, err := CreateOrganizationForCreator(context.Background(), differentExecutor, proof, "Wrong transaction")
	if created != nil || !errors.Is(err, account_model.ErrAccountUnavailable) {
		t.Fatalf("cross-transaction proof created=%v err=%v", created, err)
	}
}

func TestLockedCreatorRejectsProofWithoutTransactionOwner(t *testing.T) {
	created, err := CreateOrganizationForCreator(context.Background(), &db.Queries{}, LockedCreator{personID: 17}, "Orphaned proof")
	if created != nil || !errors.Is(err, account_model.ErrAccountUnavailable) {
		t.Fatalf("unowned proof created=%v err=%v", created, err)
	}
}

func TestCreatorLockRejectsNonTransactionalExecutor(t *testing.T) {
	// Queries created from a pool run statements in independent implicit
	// transactions; a row lock would be released before the owner write.
	_, err := LockCreator(context.Background(), &db.Queries{}, 17)
	if !errors.Is(err, db.ErrTransactionRequired) {
		t.Fatalf("pool-backed executor acquired owner lock: %v", err)
	}
}

func TestCreatorWriteRejectsProofOutsideActiveTransaction(t *testing.T) {
	q := &db.Queries{}
	proof := LockedCreator{personID: 17, executor: q}
	created, err := CreateOrganizationForCreator(context.Background(), q, proof, "No transaction")
	if created != nil || !errors.Is(err, db.ErrTransactionRequired) {
		t.Fatalf("unlocked owner write created=%v err=%v", created, err)
	}
}
