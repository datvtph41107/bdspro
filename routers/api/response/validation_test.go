package response

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestValidationContract(t *testing.T) {
	gin.SetMode(gin.TestMode)

	type input struct {
		FullName string `json:"full_name" binding:"required"`
		PersonID int64  `json:"person_id" binding:"required,min=1"`
	}
	tests := []struct {
		name  string
		body  string
		code  string
		field string
		rule  string
	}{
		{name: "missing fields use JSON tags", body: `{}`, code: "validation_failed", field: "full_name", rule: "required"},
		{name: "required numeric field", body: `{"full_name":"Jane"}`, code: "validation_failed", field: "person_id", rule: "required"},
		{name: "invalid numeric field", body: `{"full_name":"Jane","person_id":-1}`, code: "validation_failed", field: "person_id", rule: "min"},
		{name: "wrong type", body: `{"full_name":"Jane","person_id":"wrong"}`, code: "validation_failed", field: "person_id", rule: "invalid_type"},
		{name: "malformed JSON", body: `{"full_name":`, code: "malformed_json"},
		{name: "empty body", body: "", code: "request_body_required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.POST("/", func(c *gin.Context) {
				var req input
				if err := c.ShouldBindJSON(&req); err != nil {
					Validation(c, err, &req)
					return
				}
				c.Status(http.StatusNoContent)
			})
			w := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.body))
			request.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, request)

			if w.Code != http.StatusBadRequest {
				t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
			}
			var got struct {
				Error  string                  `json:"error"`
				Code   string                  `json:"code"`
				Errors map[string][]FieldIssue `json:"errors"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
				t.Fatal(err)
			}
			if got.Code != tt.code {
				t.Fatalf("code=%q want=%q body=%s", got.Code, tt.code, w.Body.String())
			}
			if tt.field != "" {
				issues := got.Errors[tt.field]
				if len(issues) == 0 || issues[0].Code != tt.rule {
					t.Fatalf("field=%q issues=%v want rule=%q", tt.field, issues, tt.rule)
				}
			}
		})
	}
}

func TestValidationDoesNotExposeBindingError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", nil)
	Validation(c, errors.New("private provider password"), struct{}{})
	if w.Code != http.StatusBadRequest || strings.Contains(w.Body.String(), "private provider password") {
		t.Fatalf("unexpected error contract: %d %s", w.Code, w.Body.String())
	}
}

func TestFieldErrorContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPatch, "/", nil)
	FieldError(c, "name", "required", "name is required")
	if w.Code != http.StatusBadRequest || c.GetString(ErrorCodeKey) != "validation_failed" {
		t.Fatalf("status=%d code=%q", w.Code, c.GetString(ErrorCodeKey))
	}
	var got struct {
		Errors map[string][]FieldIssue `json:"errors"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Errors["name"]) != 1 || got.Errors["name"][0].Code != "required" {
		t.Fatalf("unexpected field errors: %+v", got.Errors)
	}
}
