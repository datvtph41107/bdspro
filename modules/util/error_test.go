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
