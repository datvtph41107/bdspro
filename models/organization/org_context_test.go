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
