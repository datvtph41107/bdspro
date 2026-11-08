package context

import (
	"context"
	"errors"
	"time"

	auth_model "github.com/datvtph41107/bdspro/models/auth"
	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RequireSession resolves the signed first-party access principal against
// PostgreSQL before admitting an authenticated HTTP request. Token parsing
// remains owned by Authenticate and Gin still executes its native chain.
//
// A request already admitted before a concurrent revoke commits may finish.
// Operations needing commit-order authority must lock the session inside
// their own transaction; this middleware is not a transaction boundary.
func RequireSession(store interface {
	GetSession(context.Context, uuid.UUID) (db.Session, error)
}) gin.HandlerFunc {
	return func(c *gin.Context) {
		payload, ok := AuthorizationPayload(c)
		if !ok {
			AuthenticationRequired(c)
			return
		}
		if store == nil {
			accountStoreUnavailable(c, nil)
			return
		}
		err := auth_model.RequireActiveSession(c.Request.Context(), store,
			payload.AccountID, payload.SessionID, time.Now())
		switch {
		case errors.Is(err, auth_model.ErrSessionUnavailable):
			InvalidAccessToken(c)
			return
		case err != nil:
			accountStoreUnavailable(c, err)
			return
		}
		c.Next()
	}
}
