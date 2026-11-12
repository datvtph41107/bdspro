package web

import (
	"fmt"
	"net/http"
	"reflect"

	"github.com/datvtph41107/bdspro/modules/reqctx"
	"github.com/datvtph41107/bdspro/modules/web/middleware"
)

const formKey = "__form"

type bindingErrorKey struct{}

// SetForm publishes a typed request form to the shared request datastore.
func SetForm(store reqctx.ContextDataProvider, form any) {
	store.GetData()[formKey] = form
}

// GetForm requires a form previously installed by BindJSON.
func GetForm[T any](store reqctx.ContextDataProvider) T {
	value, ok := store.GetData()[formKey].(T)
	if !ok {
		panic(fmt.Sprintf("web: bound form is %T, expected %s", store.GetData()[formKey], reflect.TypeFor[T]()))
	}
	return value
}

// BindingError reports request decoding failure for the typed API action.
// Application context owns conversion of this error into the public response.
func BindingError(store reqctx.RequestDataStore) error {
	err, _ := store.GetContextValue(bindingErrorKey{}).(error)
	return err
}

// BindJSON follows Gitea's form storage ownership: decode and record the
// result, do not write an API error in the routing infrastructure.
// Typed API middleware checks BindingError and decides the HTTP response.
func BindJSON[T any]() http.HandlerFunc {
	return func(_ http.ResponseWriter, req *http.Request) {
		store := reqctx.FromContext(req.Context())
		if store == nil {
			panic("web: request datastore missing for JSON binding")
		}
		form, err := middleware.BindJSON[T](middleware.RequestJSONBinder{Request: req})
		SetForm(store, form)
		store.SetContextValue(bindingErrorKey{}, err)
	}
}
