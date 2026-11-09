package web

import (
    "fmt"
    "reflect"
)

// FormStore belongs to one HTTP request. It is not a global registry.
type FormStore interface { GetData() map[string]any }

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
