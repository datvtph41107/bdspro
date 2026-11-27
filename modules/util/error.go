package util

import "errors"

var (
	ErrInvalidArgument  = errors.New("invalid argument")
	ErrPermissionDenied = errors.New("permission denied")
	ErrNotExist         = errors.New("resource not found")
	ErrAlreadyExist     = errors.New("resource already exists")
	ErrUnauthenticated  = errors.New("authentication required")
	ErrRateLimited      = errors.New("rate limit exceeded")
)

type publicError struct {
	kind          error
	message       string
	code          string
	publicMessage string
}

func (e publicError) Error() string { return e.message }
func (e publicError) Unwrap() error { return e.kind }

// WithPublic declares a domain error with an explicit client-safe contract.
func WithPublic(kind error, modelMessage, code, publicMessage string) error {
	if kind == nil || code == "" || publicMessage == "" {
		panic("util.WithPublic: missing error category or public contract")
	}
	return publicError{
		kind:          kind,
		message:       modelMessage,
		code:          code,
		publicMessage: publicMessage,
	}
}

func ErrorForClient(err error) (category error, code, message string, ok bool) {
	var declared publicError
	if !errors.As(err, &declared) {
		return nil, "", "", false
	}
	return declared.kind, declared.code, declared.publicMessage, true
}
