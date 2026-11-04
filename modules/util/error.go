// Package util provides small shared helpers and safe error classifications.
package util

import "errors"

var (
	ErrInvalidArgument = errors.New("invalid argument")
	ErrPermissionDenied = errors.New("permission denied")
	ErrNotExist = errors.New("resource not found")
	ErrAlreadyExist = errors.New("resource already exists")
)

// kindError retains a model-specific safe message and a shared error category.
type kindError struct {
	kind error
	message string
}

func (e kindError) Error() string { return e.message }
func (e kindError) Unwrap() error { return e.kind }

// WithKind preserves a shared classification without coupling models to HTTP.
func WithKind(kind error, message string) error {
	return kindError{kind: kind, message: message}
}
