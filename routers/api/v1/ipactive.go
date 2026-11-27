package apiv1

import (
	"github.com/datvtph41107/bdspro/modules/ipactive"
	appctx "github.com/datvtph41107/bdspro/services/context"
)

func reqIPAdmission(action ipactive.Action) func(*appctx.APIContext) {
	return func(ctx *appctx.APIContext) {
		limiter, ok := ipactive.LookupMiddleware(ctx)
		if !ok {
			ctx.Admit(action, "ip", ipactive.Event{}, ipactive.ErrUnavailable)
			return
		}
		event, err := limiter.Check(ctx, action, ctx.ClientIP())
		ctx.Admit(action, "ip", event, err)
	}
}
