package auth

import (
	"context"

	"github.com/google/uuid"
)

// Actor is verified first-party session authority, never client-supplied identity.
type Actor struct {
	AccountID int64
	SessionID uuid.UUID
}

type actorContextKey struct{}

func WithActor(ctx context.Context, actor Actor) context.Context {
	if actor.AccountID <= 0 || actor.SessionID == uuid.Nil {
		panic("auth: invalid acting principal")
	}
	return context.WithValue(ctx, actorContextKey{}, actor)
}

func ActorFromContext(ctx context.Context) (Actor, bool) {
	actor, ok := ctx.Value(actorContextKey{}).(Actor)
	return actor, ok && actor.AccountID > 0 && actor.SessionID != uuid.Nil
}
