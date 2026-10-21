package source

import "errors"

var (
	ErrInvalid     = errors.New("invalid source input")
	ErrForbidden   = errors.New("source operation not permitted")
	ErrNotFound    = errors.New("source not found")
	ErrKeyConflict = errors.New("idempotency key reused with different input")
)
