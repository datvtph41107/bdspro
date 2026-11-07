package response

import appctx "github.com/datvtph41107/bdspro/services/context"

// errorResponse supports legacy tests; the classifier now has one owner.
type errorResponse struct {
	Status int
	Code string
	Message string
}

func getErrorResponse(err error) (errorResponse, bool) {
	status, code, message, ok := appctx.PublicError(err)
	if !ok {
		return errorResponse{}, false
	}
	return errorResponse{Status: status, Code: code, Message: message}, true
}
