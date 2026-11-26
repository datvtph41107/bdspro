package middleware

import "reflect"

type JSONBinder interface {
	BindJSON(value any) error
}

func BindJSON[T any](binder JSONBinder) (*T, error) {
	if reflect.TypeFor[T]().Kind() != reflect.Struct {
		panic("web/middleware: BindJSON template must be a struct")
	}
	form := new(T)
	return form, binder.BindJSON(form)
}
