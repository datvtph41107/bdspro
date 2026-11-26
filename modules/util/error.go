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

type kindError struct {
	kind    error
	message string
}

func (e kindError) Error() string { return e.message }
func (e kindError) Unwrap() error { return e.kind }

func WithKind(kind error, message string) error {
	return kindError{kind: kind, message: message}
}

func WithCause(safe, cause error) error {
	if safe == nil {
		return cause
	}
	if cause == nil {
		return safe
	}
	return safeCause{safe: safe, cause: cause}
}

type safeCause struct {
	safe  error
	cause error
}

func (e safeCause) Error() string {
	return e.safe.Error()
}

func (e safeCause) Unwrap() []error {
	return []error{e.safe, e.cause}
}

type publicError struct {
	kindError
	code          string
	publicMessage string
}

func WithPublic(kind error, modelMessage, code, publicMessage string) error {
	if kind == nil || code == "" || publicMessage == "" {
		panic("util.WithPublic: missing error category or public contract")
	}
	return publicError{
		kindError:     kindError{kind: kind, message: modelMessage},
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
