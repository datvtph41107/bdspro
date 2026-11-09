// Package middleware owns form decoding and binder-level invariants for the
// typed HTTP router. It does not publish the form or write an HTTP response.
package middleware

import "reflect"

// JSONBinder is implemented by the existing Gin-backed API request context.
// Keeping the binding capability minimal avoids a second JSON decoder.
type JSONBinder interface {
	BindJSON(value any) error
}

// BindJSON decodes a JSON object using the supplied request binder.
// Like Gitea's BindFormAny, the template must be a struct: binding an
// arbitrary scalar silently would violate the API form contract.
func BindJSON[T any](binder JSONBinder) (*T, error) {
	if reflect.TypeFor[T]().Kind() != reflect.Struct {
		panic("web/middleware: BindJSON template must be a struct")
	}
	form := new(T)
	return form, binder.BindJSON(form)
}
