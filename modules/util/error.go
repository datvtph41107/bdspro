// Package util exposes domain-neutral error categories for models and services.
package util

import "errors"

// These categories classify errors without supplying a public HTTP response.
// API responses must still match an explicit model-owned error identity.
var (
	ErrInvalidArgument  = errors.New("invalid argument")
	ErrPermissionDenied = errors.New("permission denied")
	ErrNotExist         = errors.New("resource does not exist")
	ErrAlreadyExist     = errors.New("resource already exists")
)
