package auth

import (
	"context"
	"errors"
	"testing"

	"github.com/datvtph41107/bdspro/models/auth"
	"github.com/datvtph41107/bdspro/modules/util"
	"github.com/google/uuid"
)

func TestRevokeSessionRejectsInvalidDoer(t *testing.T) {
	_, err := RevokeSession(context.Background(), nil, uuid.New())
	if !errors.Is(err, util.ErrUnauthenticated) {
		t.Fatalf("invalid doer: %v", err)
	}
	_, err = RevokeSession(context.Background(), &auth.Actor{AccountID: 42}, uuid.New())
	if !errors.Is(err, util.ErrUnauthenticated) {
		t.Fatalf("missing session authority: %v", err)
	}
}

func TestRevokeSessionRejectsInvalidTarget(t *testing.T) {
	actor := &auth.Actor{AccountID: 42, SessionID: uuid.New()}
	_, err := RevokeSession(context.Background(), actor, uuid.Nil)
	if err == nil {
		t.Fatal("nil target accepted")
	}
}
