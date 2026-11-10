package user

import (
	"net/http"

	account_model "github.com/datvtph41107/bdspro/models/account"
	db "github.com/datvtph41107/bdspro/models/db"

	api "github.com/datvtph41107/bdspro/modules/structs"
	"github.com/datvtph41107/bdspro/modules/web"
	"github.com/datvtph41107/bdspro/services/context"
	"github.com/datvtph41107/bdspro/services/convert"
	user_service "github.com/datvtph41107/bdspro/services/user"
)

// GetAuthenticatedUser exposes the verified account; request-scoped identity
// and Store are resolved in services/profile, not injected into a handler.
func GetAuthenticatedUser(ctx *context.APIContext) {
	row, err := user_service.GetAuthenticatedUser(ctx)
	if err != nil {
		ctx.APIErrorAuto(err)
		return
	}
	ctx.JSON(http.StatusOK, convert.ToAccountProfile(row))
}

// UpdateProfile consumes a typed form published by modules/web at the
// route boundary. The account ID cannot be replaced by request fields.
func UpdateProfile(ctx *context.APIContext) {
	form := web.GetForm[*api.ProfileUpdateRequest](ctx)
	row, err := user_service.UpdateProfile(ctx, user_service.ProfilePatch{
		DisplayName: form.DisplayName,
		FirstName:   form.FirstName,
		LastName:    form.LastName,
	})
	if err != nil {
		ctx.APIErrorAuto(err)
		return
	}
	ctx.JSON(http.StatusOK, convert.ToAccountProfile(row))
}

// GetConnections uses the same verified session-bound Account context as the
// other user actions. No Gin handler, handler-owned Store or client-supplied
// AccountID can select the data exposed by this endpoint.
func GetConnections(ctx *context.APIContext) {
	actor, ok := ctx.VerifiedActor()
	if !ok {
		ctx.AuthenticationRequired()
		return
	}
	store, err := db.StoreFromContext(ctx)
	if err != nil {
		ctx.APIErrorAuto(err)
		return
	}
	connections, err := account_model.GetConnections(ctx, store, actor.AccountID)
	if err != nil {
		ctx.APIErrorAuto(err)
		return
	}
	ctx.JSON(http.StatusOK, map[string]any{"connections": convert.ToConnections(connections)})
}
