package apiv1

import (
	"net/http"

	"github.com/datvtph41107/bdspro/modules/web"
	appctx "github.com/datvtph41107/bdspro/services/context"
)

// Routes exposes only the infrastructure probe during the domain reset.
func Routes() *web.Router {
	m := web.NewRouter()
	m.AfterRouting(appctx.APIContexter())
	m.Get("/health", Health)
	return m
}

func Health(ctx *appctx.APIContext) {
	ctx.JSON(http.StatusOK, map[string]string{"status": "ok"})
}
