package common

import (
	"errors"
	"strings"

	"github.com/datvtph41107/bdspro/modules/util"
	auth_service "github.com/datvtph41107/bdspro/services/auth"
	appctx "github.com/datvtph41107/bdspro/services/context"
)

func APIAuth(ctx *appctx.APIContext) {
	if ctx.Doer != nil {
		return
	}
	parts := strings.Fields(ctx.CurrentRequest().Header.Get("Authorization"))
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		ctx.AuthenticationRequired()
		return
	}
	doer, err := auth_service.ResolveAccessToken(ctx, parts[1])
	if errors.Is(err, util.ErrUnauthenticated) {
		ctx.AuthenticationRequired()
		return
	}
	if err != nil {
		ctx.APIErrorInternal(err)
		return
	}
	ctx.Doer = doer
	ctx.IsSigned = true
}
