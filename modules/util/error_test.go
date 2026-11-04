package util

import (
	"errors"
	"testing"
)

func TestWithKind(t *testing.T) {
	err := WithKind(ErrAlreadyExist, "team already exists")
	if err.Error() != "team already exists" || !errors.Is(err, ErrAlreadyExist) {
		t.Fatalf("error classification lost: %v", err)
	}
	if errors.Is(err, ErrNotExist) {
		t.Fatal("unrelated category matched")
	}
}

func TestWithCauseKeepsSafeMessageAndErrorIdentity(t *testing.T) {
	type driverError struct{ details string }
	cause := errors.New("postgres://root:secret@localhost:5432")
	public := WithKind(ErrAlreadyExist, "email already registered")
	err := WithCause(public, cause)
	if err.Error() != "email already registered" {
		t.Fatalf("sensitive database error leaked: %q", err.Error())
	}
	if !errors.Is(err, ErrAlreadyExist) || !errors.Is(err, public) || !errors.Is(err, cause) {
		t.Fatalf("error identity lost: %v", err)
	}
	if WithCause(public, nil) != public {
		t.Fatal("nil cause should preserve classification")
	}
	if WithCause(nil, cause) != cause {
		t.Fatal("nil safe error should preserve cause")
	}
	_ = driverError{}
}
