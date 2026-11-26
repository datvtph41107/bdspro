package user

import (
	"errors"
	"net/http"

	"github.com/datvtph41107/bdspro/modules/util"
	authhttp "github.com/datvtph41107/bdspro/routers/api/v1/auth"
	authservice "github.com/datvtph41107/bdspro/services/auth"
	appctx "github.com/datvtph41107/bdspro/services/context"
	"github.com/google/uuid"
)

func ListSessions(ctx *appctx.APIContext) {
	ctx.Header("Cache-Control", "no-store")
	if ctx.Doer == nil {
		ctx.AuthenticationRequired()
		return
	}
	items, err := authservice.ListSessions(ctx, *ctx.Doer)
	if errors.Is(err, util.ErrUnauthenticated) {
		ctx.AuthenticationRequired()
		return
	}
	if err != nil {
		ctx.APIErrorAuto(err)
		return
	}
	ctx.JSON(http.StatusOK, map[string]any{"items": items})
}

func RevokeSession(ctx *appctx.APIContext) {
	ctx.Header("Cache-Control", "no-store")
	if ctx.Doer == nil {
		ctx.AuthenticationRequired()
		return
	}
	id, err := uuid.Parse(ctx.PathParam("session"))
	if err != nil || id == uuid.Nil || id.String() != ctx.PathParam("session") {
		ctx.APIError(http.StatusBadRequest, "invalid_request", "invalid session identifier")
		return
	}
	err = authservice.RevokeSession(ctx, *ctx.Doer, id)
	if errors.Is(err, util.ErrUnauthenticated) {
		ctx.AuthenticationRequired()
		return
	}
	if err != nil {
		ctx.APIErrorAuto(err)
		return
	}
	if id == ctx.Doer.SessionID {
		authhttp.ClearRefreshCookie(ctx)
	}
	ctx.Status(http.StatusNoContent)
}
