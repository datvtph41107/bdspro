// Package middleware owns HTTP form binding, independently from router and
// response handling. The transport context provides its own JSON binder.
package middleware

// JSONBinder is the minimal capability needed to decode and validate a JSON
// request. Gin supplies it through services/context, not from this package.
type JSONBinder interface {
	BindJSON(value any) error
}

// BindJSON creates one typed form. The route owns error presentation and only
// stores the form in modules/web after successful binding.
func BindJSON[T any](binder JSONBinder) (*T, error) {
	form := new(T)
	if err := binder.BindJSON(form); err != nil {
		return form, err
	}
	return form, nil
}
