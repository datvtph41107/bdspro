package response

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// FieldIssue identifies an invalid public JSON field without exposing raw input.
type FieldIssue struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// FieldErrors writes the existing {error, code} envelope with per-field details.
// HTTP 400 is preserved for compatibility with the current unversioned API.
func FieldErrors(c *gin.Context, issues map[string][]FieldIssue) {
	c.Set(ErrorCodeKey, "validation_failed")
	c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
		"error":  "validation failed",
		"code":   "validation_failed",
		"errors": issues,
	})
}

// FieldError handles a manual, field-specific HTTP validation rule.
func FieldError(c *gin.Context, field, code, message string) {
	FieldErrors(c, map[string][]FieldIssue{
		field: {{Code: code, Message: message}},
	})
}

// Validation translates Gin binding errors into public JSON field names.
// The request is passed explicitly to read its json tags; no global validator
// registry or Gin configuration mutation is required.
func Validation(c *gin.Context, err error, request any) {
	var size *http.MaxBytesError
	if errors.As(err, &size) {
		Reject(c, http.StatusRequestEntityTooLarge, "request_body_too_large", "request body too large")
		return
	}

	var fields validator.ValidationErrors
	if errors.As(err, &fields) {
		issues := make(map[string][]FieldIssue, len(fields))
		for _, failure := range fields {
			field := jsonFieldName(request, failure.StructField())
			issues[field] = append(issues[field], FieldIssue{
				Code:    failure.Tag(),
				Message: fieldMessage(field, failure),
			})
		}
		FieldErrors(c, issues)
		return
	}

	var wrongType *json.UnmarshalTypeError
	if errors.As(err, &wrongType) {
		if wrongType.Field != "" {
			FieldError(c, wrongType.Field, "invalid_type", wrongType.Field+" has an invalid type")
			return
		}
		Reject(c, http.StatusBadRequest, "invalid_json_type", "invalid JSON value type")
		return
	}

	var syntax *json.SyntaxError
	if errors.As(err, &syntax) || errors.Is(err, io.ErrUnexpectedEOF) {
		Reject(c, http.StatusBadRequest, "malformed_json", "malformed JSON body")
		return
	}

	if errors.Is(err, io.EOF) {
		Reject(c, http.StatusBadRequest, "request_body_required", "request body is required")
		return
	}

	// Unrecognized binding failures stay private: do not serialize err.Error().
	Reject(c, http.StatusBadRequest, "invalid_request_body", "request body is invalid")
}

func jsonFieldName(request any, name string) string {
	t := reflect.TypeOf(request)
	if t == nil {
		return name
	}
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return name
	}
	field, ok := t.FieldByName(name)
	if !ok {
		return name
	}
	jsonName := strings.Split(field.Tag.Get("json"), ",")[0]
	if jsonName != "" && jsonName != "-" {
		return jsonName
	}
	return name
}

func fieldMessage(field string, failure validator.FieldError) string {
	switch failure.Tag() {
	case "required":
		return field + " is required"
	case "email":
		return field + " must be a valid email"
	case "min":
		return field + " must be at least " + failure.Param()
	case "max":
		return field + " must be at most " + failure.Param()
	default:
		return field + " is invalid"
	}
}
