package account

import (
	"errors"

	"github.com/datvtph41107/bdspro/modules/util"
)

var (
	ErrAccountUnavailable   = errors.New("account unavailable")
	ErrEmailRegistered      = util.WithKind(util.ErrAlreadyExist, "email already registered")
	ErrEmailAlreadyVerified = errors.New("email already verified")
	ErrEmailLinkUnavailable = errors.New("account cannot attach an email")
	ErrEmailLinkTaken       = errors.New("email address belongs to another account")
	ErrEmailLinkCooldown    = errors.New("email verification temporarily unavailable")
	ErrEmailLinkInvalid     = errors.New("invalid or expired email link")
)
