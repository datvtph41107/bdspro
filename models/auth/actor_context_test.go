package auth

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestActorContextIsolation(t *testing.T) {
	parent := context.Background()
	a := WithActor(parent, Actor{AccountID: 11, SessionID: uuid.New()})
	b := WithActor(parent, Actor{AccountID: 12, SessionID: uuid.New()})
	if _, ok := ActorFromContext(parent); ok {
		t.Fatal("parent unexpectedly has actor")
	}
	gotA, okA := ActorFromContext(a)
	gotB, okB := ActorFromContext(b)
	if !okA || !okB || gotA.AccountID != 11 || gotB.AccountID != 12 {
		t.Fatalf("wrong authorities: %+v %+v", gotA, gotB)
	}
}
