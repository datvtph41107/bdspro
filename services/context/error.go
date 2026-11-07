package context

import (
	"errors"
	"net/http"

	"github.com/datvtph41107/bdspro/modules/util"
)

// PublicError classifies only explicitly safe domain errors for HTTP.
func PublicError(err error) (status int, code, message string, ok bool) {
	category, code, message, ok := util.ErrorForClient(err)
	if !ok {
		return 0, "", "", false
	}
		switch {
	case errors.Is(category, util.ErrInvalidArgument):
		status = http.StatusBadRequest
	case errors.Is(category, util.ErrUnauthenticated):
		status = http.StatusUnauthorized
	case errors.Is(category, util.ErrPermissionDenied):
		status = http.StatusForbidden
	case errors.Is(category, util.ErrNotExist):
		status = http.StatusNotFound
	case errors.Is(category, util.ErrAlreadyExist):
		status = http.StatusConflict
	case errors.Is(category, util.ErrRateLimited):
		status = http.StatusTooManyRequests
	default:
		return 0, "", "", false
	}
	return status, code, message, true
}
