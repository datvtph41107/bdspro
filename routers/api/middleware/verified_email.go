package middleware

import (
	"context"

	db "github.com/datvtph41107/bdspro/models/db"
	appctx "github.com/datvtph41107/bdspro/services/context"
	"github.com/gin-gonic/gin"
)

// VerifiedEmail delegates admission to the canonical context boundary.
func VerifiedEmail(store interface {
	GetAccountByID(context.Context, int64) (db.Account, error)
}) gin.HandlerFunc {
	return appctx.VerifiedEmail(store)
}

// VerifiedContact delegates admission without changing query or failure policy.
func VerifiedContact(store interface {
	GetAccountByID(context.Context, int64) (db.Account, error)
	GetPhoneForAccount(context.Context, int64) (string, error)
}) gin.HandlerFunc {
	return appctx.VerifiedContact(store)
}
