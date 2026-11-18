// Package auth implements authentication orchestration, keeping JWT
// cryptography in modules/auth and session persistence in models/db.
package auth

import (
	"context"

	authmodel "github.com/datvtph41107/bdspro/models/auth"
	token "github.com/datvtph41107/bdspro/modules/auth"
	"github.com/datvtph41107/bdspro/modules/util"
	"github.com/google/uuid"
)

// ResolveAccessToken requires both a valid access JWT and a live matching
// PostgreSQL session. It shares one session authority with browser login.
func ResolveAccessToken(ctx context.Context, encoded string) (*authmodel.Actor, error) {
	payload, err := token.ProcessMaker().VerifyToken(encoded, token.TokenTypeAccess)
	if err != nil || payload.SessionID == uuid.Nil {
		return nil, util.ErrUnauthenticated
	}

	actor, err := ResolveSessionActor(ctx, payload.SessionID)
	if err != nil {
		return nil, err
	}
	if actor.AccountID != payload.AccountID {
		return nil, util.ErrUnauthenticated
	}
	return actor, nil
}
