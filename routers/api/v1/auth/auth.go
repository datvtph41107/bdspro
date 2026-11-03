// Package auth holds HTTP admission policies. Persistent authorization
// invariants remain in model/SQL operations.
package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"

	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/modules/token"
	apictx "github.com/datvtph41107/bdspro/services/context"
	"github.com/jackc/pgx/v5"
)

// Authenticate is a typed Gitea-style route middleware. It is not global:
// public registration, email verification, OAuth callback and reset routes
// must continue to work without a Bearer token.
func Authenticate(maker token.Maker) func(*apictx.APIContext) {
	return func(ctx *apictx.APIContext) {
		parts := strings.Fields(ctx.Req.Header.Get("Authorization"))
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			ctx.BearerRequired()
			return
		}
		payload, err := maker.VerifyToken(parts[1], token.TokenTypeAccess)
		if err != nil {
			ctx.InvalidAccessToken()
			return
		}
		ctx.Doer = &apictx.Principal{AccountID: payload.AccountID}
	}
}

type ContactReader interface {
	GetAccountByID(context.Context, int64) (db.Account, error)
	GetPhoneForAccount(context.Context, int64) (string, error)
}

func VerifiedContact(store ContactReader) func(*apictx.APIContext) {
	return func(ctx *apictx.APIContext) {
		if store == nil {
			ctx.IdentityStoreUnavailable()
			return
		}
		if ctx.Doer == nil {
			ctx.AuthenticationRequired()
			return
		}
		account, err := store.GetAccountByID(ctx, ctx.Doer.AccountID)
		if errors.Is(err, pgx.ErrNoRows) {
			ctx.AuthenticationRequired()
			return
		}
		if err != nil {
			ctx.IdentityStoreUnavailable()
			return
		}
		if account.IsEmailVerified {
			return
		}
		_, err = store.GetPhoneForAccount(ctx, account.ID)
		if err == nil {
			return
		}
		if errors.Is(err, pgx.ErrNoRows) {
			ctx.APIError(http.StatusForbidden, "contact_verification_required", "verified contact required")
			return
		}
		ctx.IdentityStoreUnavailable()
	}
}

func VerifiedEmail(store interface {
	GetAccountByID(context.Context, int64) (db.Account, error)
}) func(*apictx.APIContext) {
	return func(ctx *apictx.APIContext) {
		if store == nil {
			ctx.IdentityStoreUnavailable()
			return
		}
		if ctx.Doer == nil {
			ctx.AuthenticationRequired()
			return
		}
		account, err := store.GetAccountByID(ctx, ctx.Doer.AccountID)
		if errors.Is(err, pgx.ErrNoRows) {
			ctx.AuthenticationRequired()
			return
		}
		if err != nil {
			ctx.IdentityStoreUnavailable()
			return
		}
		if !account.IsEmailVerified {
			ctx.APIError(http.StatusForbidden, "email_verification_required", "email verification required")
		}
	}
}
