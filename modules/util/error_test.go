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
}

func TestErrorForClientExplicitOptInOnly(t *testing.T) {
	const secret = "postgres://user:privatepassword@internal"
	public := WithPublic(ErrNotExist, "private model context", "organization.team_not_found", "team not found")
	wrapped := WithCause(public, errors.New(secret))
	kind, code, message, ok := ErrorForClient(wrapped)
	if !ok || !errors.Is(kind, ErrNotExist) || code != "organization.team_not_found" || message != "team not found" {
		t.Fatalf("explicit public metadata lost: category=%v code=%q message=%q ok=%t", kind, code, message, ok)
	}
	if wrapped.Error() != "private model context" || errors.Is(wrapped, ErrAlreadyExist) {
		t.Fatalf("unsafe model error or incorrect classification: %v", wrapped)
	}
	for _, err := range []error{errors.New(secret), ErrNotExist, WithKind(ErrNotExist, "not owned")} {
		_, _, _, ok := ErrorForClient(err)
		if ok {
			t.Fatalf("unowned error was exposed: %v", err)
		}
	}
}

func TestErrorForClientThroughJoin(t *testing.T) {
	declared := WithPublic(ErrAlreadyExist, "private conflict", "resource.conflict", "already exists")
	kind, code, msg, ok := ErrorForClient(errors.Join(declared, errors.New("sensitive driver detail")))
	if !ok || !errors.Is(kind, ErrAlreadyExist) || code != "resource.conflict" || msg != "already exists" {
		t.Fatalf("lost public metadata through Join: %v %s %s %v", kind, code, msg, ok)
	}
}
