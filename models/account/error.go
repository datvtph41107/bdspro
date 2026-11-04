package account

import "errors"

var (
	ErrAccountUnavailable = errors.New("account unavailable")
	ErrEmailRegistered    = errors.New("email already registered")
)
