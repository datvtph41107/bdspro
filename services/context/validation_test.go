package context

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/datvtph41107/bdspro/modules/web"
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
			router := web.NewRouter()
			router.AfterRouting(APIContexter())
			router.Post("/", func(ctx *APIContext) {
				var req input
				if err := ctx.BindJSON(&req); err != nil {
					ctx.Validation(err, &req)
					return
				}
				ctx.Status(http.StatusNoContent)
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
	router := web.NewRouter()
	router.AfterRouting(APIContexter())
	router.Post("/", func(ctx *APIContext) {
		ctx.Validation(errors.New("private provider password"), struct{}{})
	})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/", nil))
	if w.Code != http.StatusBadRequest || strings.Contains(w.Body.String(), "private provider password") {
		t.Fatalf("unexpected error contract: %d %s", w.Code, w.Body.String())
	}
}

func TestFieldErrorContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := web.NewRouter()
	router.AfterRouting(APIContexter())
	router.Patch("/", func(ctx *APIContext) {
		ctx.FieldError("name", "required", "name is required")
	})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodPatch, "/", nil))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var got struct {
		Code   string                  `json:"code"`
		Errors map[string][]FieldIssue `json:"errors"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Code != "validation_failed" || len(got.Errors["name"]) != 1 || got.Errors["name"][0].Code != "required" {
		t.Fatalf("unexpected field errors: %+v", got)
	}
}
