package web

import (
	"fmt"
	"reflect"

	"github.com/datvtph41107/bdspro/modules/reqctx"
)

// FormStore is exactly the request-data capability created by modules/reqctx.
// No second form registry or global map is owned by modules/web.
type FormStore = reqctx.ContextDataProvider

const formKey = "__form"

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
