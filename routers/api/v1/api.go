// Package apiv1 declares the HTTP surface in one Gitea-style route table.
// This initial implementation is an UNMOUNTED organization candidate; it must
// not replace R01's production router until the other 28 endpoints migrate.
package apiv1

import (
	"net/http"
	"sync"

	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/modules/reqctx"
	"github.com/datvtph41107/bdspro/modules/token"
	"github.com/datvtph41107/bdspro/modules/web"
	"github.com/datvtph41107/bdspro/routers/api/v1/auth"
	"github.com/datvtph41107/bdspro/routers/api/v1/org"
	apictx "github.com/datvtph41107/bdspro/services/context"
)

var registerAPIContextOnce sync.Once

// Routes prepares a separate candidate for HTTP contract testing, not a
// parallel production router. Mount at root after all endpoints are migrated.
func Routes(store db.Store, maker token.Maker) *web.Router {
	registerAPIContextOnce.Do(func() { web.RegisterResponseStatusProvider[*apictx.APIContext](apictx.GetAPIContext) })
	m := web.NewRouter()
	m.BeforeRouting(reqctx.RequestContextHandler())
	m.AfterRouting(apictx.APIContexter())
	m.Get("/health", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	orgAPI := org.NewAPI(store)
	m.Group("/organizations", func() {
		m.Get("", orgAPI.List)
		m.Post("", orgAPI.Create)
		m.Group("/{id}", func() {
			m.Get("", orgAPI.Get)
			m.Patch("", orgAPI.Edit)
			m.Get("/members", orgAPI.Members)
			m.Post("/members", orgAPI.AddMember)
		})
	}, auth.Authenticate(maker), auth.VerifiedContact(store))
	return m
}

