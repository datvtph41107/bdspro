package middleware

import (
	"context"
	"errors"
	"github.com/datvtph41107/bdspro/internal/api/response"
	"github.com/jackc/pgx/v5"
	"net/http"

	db "github.com/datvtph41107/bdspro/internal/database"
	"github.com/gin-gonic/gin"
)

// VerifiedEmail guards business operations, while still allowing unverified
// users to authenticate, inspect their own account and verify their email.
func VerifiedEmail(store interface {
	GetAccountByID(context.Context, int64) (db.Account, error)
}) gin.HandlerFunc {
	return func(c *gin.Context) {
		if store == nil {
			response.Reject(c, http.StatusServiceUnavailable, "identity_store_unavailable", "identity store unavailable")
			return
		}
		account, err := store.GetAccountByID(c.Request.Context(), AccountID(c))
		if errors.Is(err, pgx.ErrNoRows) {
			response.Reject(c, http.StatusUnauthorized, "unauthenticated", "authentication required")
			return
		}
		if err != nil {
			response.Reject(c, http.StatusServiceUnavailable, "identity_store_unavailable", "identity store unavailable")
			return
		}
		if !account.IsEmailVerified {
			response.Reject(c, http.StatusForbidden, "email_verification_required", "email verification required")
			return
		}
		c.Next()
	}
}

 
// VerifiedContact requires at least one verified contact channel. A phone-only
// account must not be excluded from business routes merely for lacking email.
func VerifiedContact(store interface {
 GetAccountByID(context.Context, int64) (db.Account, error)
 GetPhoneForAccount(context.Context, int64) (string, error)
}) gin.HandlerFunc {
 return func(c *gin.Context) {
  a, err := store.GetAccountByID(c.Request.Context(), AccountID(c))
  if errors.Is(err, pgx.ErrNoRows) {
   response.Reject(c, http.StatusUnauthorized, "unauthenticated", "authentication required")
   return
  }
  if err != nil {
   response.Reject(c, http.StatusServiceUnavailable, "identity_store_unavailable", "identity store unavailable")
   return
  }
  if a.IsEmailVerified { c.Next(); return }
  _, err = store.GetPhoneForAccount(c.Request.Context(), a.ID)
  if err == nil { c.Next(); return }
  if errors.Is(err, pgx.ErrNoRows) {
   response.Reject(c, http.StatusForbidden, "contact_verification_required", "verified contact required")
   return
  }
  response.Reject(c, http.StatusServiceUnavailable, "identity_store_unavailable", "identity store unavailable")
 }
}
