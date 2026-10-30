package middleware

import (
	"context"
	"errors"
	"fmt"
	"github.com/datvtph41107/bdspro/internal/api/response"
	"github.com/jackc/pgx/v5"

	db "github.com/datvtph41107/bdspro/internal/database"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// VerifiedEmail guards business operations, while still allowing unverified
// users to authenticate, inspect their own account and verify their email.
func VerifiedEmail(store interface {
	GetAccountByID(context.Context, int64) (db.Account, error)
}) gin.HandlerFunc {
	return func(c *gin.Context) {
		if store == nil {
			identityStoreUnavailable(c, nil)
			return
		}
		account, err := store.GetAccountByID(c.Request.Context(), AccountID(c))
		if errors.Is(err, pgx.ErrNoRows) {
			response.Fail(c, response.AuthenticationRequired)
			return
		}
		if err != nil {
			identityStoreUnavailable(c, err)
			return
		}
		if !account.IsEmailVerified {
			response.Fail(c, response.EmailVerificationRequired)
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
		if store == nil {
			identityStoreUnavailable(c, nil)
			return
		}
		a, err := store.GetAccountByID(c.Request.Context(), AccountID(c))
		if errors.Is(err, pgx.ErrNoRows) {
			response.Fail(c, response.AuthenticationRequired)
			return
		}
		if err != nil {
			identityStoreUnavailable(c, err)
			return
		}
		if a.IsEmailVerified {
			c.Next()
			return
		}
		_, err = store.GetPhoneForAccount(c.Request.Context(), a.ID)
		if err == nil {
			c.Next()
			return
		}
		if errors.Is(err, pgx.ErrNoRows) {
			response.Fail(c, response.ContactVerificationRequired)
			return
		}
		identityStoreUnavailable(c, err)
	}
}

func identityStoreUnavailable(c *gin.Context, err error) {
	event := log.Ctx(c.Request.Context()).Error().Str("dependency", "postgres")
	if err != nil {
		// The raw database error may contain credentials or private data.
		event = event.Str("error_type", fmt.Sprintf("%T", err))
	}
	event.Msg("auth.identity_store_unavailable")
	response.Fail(c, response.IdentityStoreUnavailable)
}
