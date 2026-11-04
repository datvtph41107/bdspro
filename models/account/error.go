package account

import (
	"errors"

	"github.com/datvtph41107/bdspro/modules/util"
)

var (
	ErrAccountUnavailable = errors.New("account unavailable")
	ErrEmailRegistered    = util.WithKind(util.ErrAlreadyExist, "email already registered")
)
