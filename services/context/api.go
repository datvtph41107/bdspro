// Package context owns the HTTP request context and response contract.
package context

import (
	"context"

	auth_model "github.com/datvtph41107/bdspro/models/auth"
	authcrypto "github.com/datvtph41107/bdspro/modules/auth"
)

// APIContext is the API-specific request state. It does not embed or import
// Gin. The only Gin boundary is gin.go, which builds and reuses this context.
type APIContext struct {
	*Base

	// Doer is signed JWT claims, not an Account model and not write authority.
	// IsSigned becomes true only after PostgreSQL session verification.
	Doer *authcrypto.Payload
	IsSigned bool
	Org *APIOrganization
}

type APIHandler func(*APIContext)

var _ context.Context = (*APIContext)(nil)

// VerifiedActor is the sole HTTP-to-domain verified principal accessor.
func (c *APIContext) VerifiedActor() (auth_model.Actor, bool) {
	if c == nil || c.Base == nil || !c.IsSigned || c.Doer == nil {
		return auth_model.Actor{}, false
	}
	actor, ok := auth_model.ActorFromContext(c)
	if !ok || actor.AccountID != c.Doer.AccountID || actor.SessionID != c.Doer.SessionID {
		return auth_model.Actor{}, false
	}
	return actor, true
}
