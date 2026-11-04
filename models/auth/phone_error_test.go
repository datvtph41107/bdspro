package auth

import (
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestPhoneConstraintClassification(t *testing.T) {
	phone := &pgconn.PgError{Code: "23505", ConstraintName: "phone_credentials_phone_e164_key"}
	if !phoneRegisteredConflict(phone) || !phoneCredentialConflict(phone) {
		t.Fatal("phone uniqueness not recognized")
	}
	owner := &pgconn.PgError{Code: "23505", ConstraintName: "phone_credentials_pkey"}
	if phoneRegisteredConflict(owner) || !phoneCredentialConflict(owner) {
		t.Fatal("account-key conflict must only apply to binding")
	}
	unrelated := &pgconn.PgError{Code: "23505", ConstraintName: "some_other_pkey"}
	if phoneRegisteredConflict(unrelated) || phoneCredentialConflict(unrelated) {
		t.Fatal("unrelated unique index must remain unexpected")
	}
}
