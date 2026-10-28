package middleware

import (
	"context"
	"net/http"

	db "github.com/datvtph41107/bdspro/internal/db/sqlc"
	"github.com/gin-gonic/gin"
)

// VerifiedEmail guards business operations, while still allowing unverified
// users to authenticate, inspect their own account and verify their email.
func VerifiedEmail(store interface {
	GetAccountByID(context.Context, int64) (db.Account, error)
}) gin.HandlerFunc {
	return func(c *gin.Context) {
		if store == nil {
			c.AbortWithStatus(http.StatusServiceUnavailable)
			return
		}
		account, err := store.GetAccountByID(c.Request.Context(), AccountID(c))
		if err != nil {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		if !account.IsEmailVerified {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "email verification required"})
			return
		}
		c.Next()
	}
}
