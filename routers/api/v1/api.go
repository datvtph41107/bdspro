package apiv1

import (
	"net/http"

	"github.com/datvtph41107/bdspro/modules/web"
	"github.com/datvtph41107/bdspro/modules/structs"
	org "github.com/datvtph41107/bdspro/routers/api/v1/org"
	"github.com/datvtph41107/bdspro/routers/common"
	"github.com/datvtph41107/bdspro/modules/web/middleware"
	appctx "github.com/datvtph41107/bdspro/services/context"
)

// Routes composes the health probe and Organization/Team domain API.
func Routes() *web.Router {
	m := web.NewRouter()
	m.AfterRouting(appctx.APIContexter())
	m.Get("/health", Health)
	m.Group("/orgs/{org}", func() {
		m.Combo("/teams").
			Get(org.ListTeams).
			Post(reqOrgOwner(), bind(structs.CreateTeamOption{}), org.CreateTeam)
		m.Combo("/teams/{team}").
			Get(org.GetTeam).
			Patch(reqOrgOwner(), bind(structs.RenameTeamOption{}), org.RenameTeam)
	}, common.APIAuth, orgAssignment)
	return m
}

// bind creates a typed route action that decodes and validates a JSON form
// before the endpoint runs. Gitea's API router uses the same local bind
// helper pattern, keeping HTTP error handling out of modules/web/middleware.
//
// tmpl declares the desired form type at route registration time.
// A successful bind publishes *T to the current request's datastore.
func bind[T any](tmpl T) any {
	return func(ctx *appctx.APIContext) {
		form, err := middleware.BindJSON[T](ctx)
		if err != nil {
			ctx.Validation(err, form)
			return
		}
		web.SetForm(ctx, form)
	}
}

func Health(ctx *appctx.APIContext) {
	ctx.JSON(http.StatusOK, map[string]string{"status": "ok"})
}
