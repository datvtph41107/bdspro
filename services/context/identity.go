package context

import (
	"errors"
	"fmt"
	"net/http"

	account_model "github.com/datvtph41107/bdspro/models/account"
	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/modules/log"
	"github.com/jackc/pgx/v5"
)

// VerifiedEmail applies the domain admission policy to an already
// session-verified actor. The request context provides the sole DB executor.
// This is a typed Gitea-style route guard, not a Gin handler.
func VerifiedEmail() func(*APIContext) {
	return func(ctx *APIContext) {
		actor, ok := ctx.VerifiedActor()
		if !ok {
			ctx.AuthenticationRequired()
			return
		}
		_, err := db.GetQueries(ctx)
		if err != nil {
			identityStoreUnavailable(ctx, err)
			return
		}
		account, err := account_model.ReadByID(ctx, actor.AccountID)
		switch {
		case errors.Is(err, account_model.ErrAccountUnavailable):
			ctx.AuthenticationRequired()
			return
		case err != nil:
			identityStoreUnavailable(ctx, err)
			return
		case !account.IsEmailVerified:
			ctx.APIError(http.StatusForbidden, "email_verification_required", "email verification required")
		}
	}
}

// VerifiedContact admits accounts with either a verified email address or a
// bound phone. Its policy is evaluated from persisted state, never JWT claims.
func VerifiedContact() func(*APIContext) {
	return func(ctx *APIContext) {
		actor, ok := ctx.VerifiedActor()
		if !ok {
			ctx.AuthenticationRequired()
			return
		}
		_, err := db.GetQueries(ctx)
		if err != nil {
			identityStoreUnavailable(ctx, err)
			return
		}
		account, err := account_model.ReadByID(ctx, actor.AccountID)
		if errors.Is(err, account_model.ErrAccountUnavailable) {
			ctx.AuthenticationRequired()
			return
		}
		if err != nil {
			identityStoreUnavailable(ctx, err)
			return
		}
		if account.IsEmailVerified {
			return
		}
		q, getErr := db.GetQueries(ctx)
		if getErr != nil {
			identityStoreUnavailable(ctx, getErr)
			return
		}
		_, err = q.GetPhoneForAccount(ctx, account.ID)
		if err == nil {
			return
		}
		if errors.Is(err, pgx.ErrNoRows) {
			ctx.APIError(http.StatusForbidden, "contact_verification_required", "verified contact required")
			return
		}
		identityStoreUnavailable(ctx, err)
	}
}

func identityStoreUnavailable(ctx *APIContext, err error) {
	event := log.Ctx(ctx).Error().Str("dependency", "postgres")
	if err != nil {
		event = event.Str("error_type", fmt.Sprintf("%T", err))
	}
	event.Msg("auth.identity_store_unavailable")
	ctx.APIError(http.StatusServiceUnavailable, "identity_store_unavailable", "identity store unavailable")
}
