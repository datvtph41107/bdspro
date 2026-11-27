package context

import (
	"context"
	"net/http"

	auth_model "github.com/datvtph41107/bdspro/models/auth"
	"github.com/datvtph41107/bdspro/modules/web"
	web_types "github.com/datvtph41107/bdspro/modules/web/types"
)

type apiContextRequestKey struct{}

func GetAPIContextFromRequest(req *http.Request) *APIContext {
	api, ok := req.Context().Value(apiContextRequestKey{}).(*APIContext)
	if !ok || api == nil {
		panic("context: APIContext missing from HTTP request")
	}

	if source, ok := api.Base.source.(*httpRequestSource); ok {
		source.request = req
	}
	return api
}

func init() {
	web.RegisterResponseStatusProvider[*APIContext](func(req *http.Request) web_types.ResponseStatusProvider {
		return GetAPIContextFromRequest(req)
	})
}

type APIContext struct {
	*Base

	Doer     *auth_model.Actor
	IsSigned bool
	Org      *APIOrganization
}

var _ context.Context = (*APIContext)(nil)
var _ web_types.ResponseStatusProvider = (*APIContext)(nil)
