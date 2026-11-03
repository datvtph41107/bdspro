package context

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

type FieldIssue struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (ctx *APIContext) FieldError(field, code, message string) {
	ctx.FieldErrors(map[string][]FieldIssue{
		field: {{Code: code, Message: message}},
	})
}

func (ctx *APIContext) FieldErrors(issues map[string][]FieldIssue) {
	ctx.JSON(http.StatusBadRequest, struct {
		Error  string                  `json:"error"`
		Code   string                  `json:"code"`
		Errors map[string][]FieldIssue `json:"errors"`
	}{Error: "validation failed", Code: "validation_failed", Errors: issues})
}

// BindJSON maintains R01's lenient JSON-field policy and compatibility for
// invalid bodies. Required-field validation remains explicit in endpoints.
func (ctx *APIContext) BindJSON(target any) bool {
	if ctx.Req.Body == nil {
		ctx.APIError(http.StatusBadRequest, "request_body_required", "request body is required")
		return false
	}
	decoder := json.NewDecoder(ctx.Req.Body)
	if err := decoder.Decode(target); err != nil {
		var max *http.MaxBytesError
		var wrong *json.UnmarshalTypeError
		var syntax *json.SyntaxError
		switch {
		case errors.As(err, &max):
			ctx.APIError(http.StatusRequestEntityTooLarge, "request_body_too_large", "request body too large")
		case errors.As(err, &wrong) && wrong.Field != "":
			ctx.FieldError(wrong.Field, "invalid_type", wrong.Field+" has an invalid type")
		case errors.As(err, &syntax) || errors.Is(err, io.ErrUnexpectedEOF):
			ctx.APIError(http.StatusBadRequest, "malformed_json", "malformed JSON body")
		case errors.Is(err, io.EOF):
			ctx.APIError(http.StatusBadRequest, "request_body_required", "request body is required")
		default:
			ctx.APIError(http.StatusBadRequest, "invalid_request_body", "request body is invalid")
		}
		return false
	}
	var remaining any
	if err := decoder.Decode(&remaining); !errors.Is(err, io.EOF) {
		ctx.APIError(http.StatusBadRequest, "invalid_request_body", "request body is invalid")
		return false
	}
	return true
}
