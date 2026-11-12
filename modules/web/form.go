package web

import (
	"fmt"
	"reflect"

	"github.com/datvtph41107/bdspro/modules/reqctx"
)

const formKey = "__form"

// SetForm publishes a decoded request form. The caller must validate and
// reject malformed input before publishing the form.
func SetForm(store reqctx.ContextDataProvider, form any) {
	store.GetData()[formKey] = form
}

func GetForm[T any](store reqctx.ContextDataProvider) T {
	value, ok := store.GetData()[formKey].(T)
	if !ok {
		panic(fmt.Sprintf("web: bound form is %T, expected %s", store.GetData()[formKey], reflect.TypeFor[T]()))
	}
	return value
}
