package web

import (
	"fmt"
	"reflect"

	"github.com/datvtph41107/bdspro/modules/reqctx"
	"github.com/datvtph41107/bdspro/modules/web/middleware"
)

// FormStore is exactly the request-data capability created by modules/reqctx.
// No second form registry or global map is owned by modules/web.
type FormStore = reqctx.ContextDataProvider

const formKey = "__form"

// JSONFormContext is the minimal request-scoped contract used when the same
// request binds JSON and stores its typed form. No HTTP domain dependencies.
type JSONFormContext interface {
	FormStore
	middleware.JSONBinder
}

// BindJSON binds once, then publishes the form only on success. The HTTP
// route still owns how binding errors become public API validation responses.
func BindJSON[T any](ctx JSONFormContext) (*T, error) {
	form, err := middleware.BindJSON[T](ctx)
	if err != nil {
		return form, err
	}
	SetForm(ctx, form)
	return form, nil
}

func SetForm(store FormStore, value any) {
	store.GetData()[formKey] = value
}

func GetForm[T any](store FormStore) T {
	value := store.GetData()[formKey]
	form, ok := value.(T)
	if !ok {
		panic(fmt.Errorf("web: bound form %T does not match requested %s", value, reflect.TypeFor[T]()))
	}
	return form
}
