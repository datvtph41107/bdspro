package response

import (
	"errors"
	"net/http"

	"github.com/datvtph41107/bdspro/modules/util"
)

// errorResponse contains the HTTP status and safe client message for a Go error.
type errorResponse struct {
	Status  int
	Code    string
	Message string
}

// getErrorResponse exposes only explicitly declared model failures.
// Models own stable client identities; the HTTP boundary maps error categories
// to transport status codes, without a registry of domain errors.
func getErrorResponse(err error) (errorResponse, bool) {
	category, code, message, ok := util.ErrorForClient(err)
	if !ok {
		return errorResponse{}, false
	}
	var status int
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
		return errorResponse{}, false
	}
	return errorResponse{Status: status, Code: code, Message: message}, true
}
