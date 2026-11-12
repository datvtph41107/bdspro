// Package middleware owns request decoding independently of Gin.
package middleware

import (
	"errors"
	"io"
	"net/http"
	"reflect"

	appjson "github.com/datvtph41107/bdspro/modules/json"
)

// JSONBinder is the transport-agnostic binding capability.
type JSONBinder interface {
	BindJSON(value any) error
}

// RequestJSONBinder decodes one JSON document using the application codec.
// The caller owns the HTTP error response and request body size policy.
type RequestJSONBinder struct {
	Request *http.Request
}

func (b RequestJSONBinder) BindJSON(value any) error {
	if b.Request == nil || b.Request.Body == nil {
		return errors.New("web/middleware: missing request body")
	}
	decoder := appjson.NewDecoder(b.Request.Body)
	if err := decoder.Decode(value); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return errors.New("web/middleware: multiple JSON values")
		}
		return err
	}
	return nil
}

// BindJSON creates a typed form and delegates the decoding to the binder.
func BindJSON[T any](binder JSONBinder) (*T, error) {
	if reflect.TypeFor[T]().Kind() != reflect.Struct {
		panic("web/middleware: BindJSON template must be a struct")
	}
	form := new(T)
	return form, binder.BindJSON(form)
}
