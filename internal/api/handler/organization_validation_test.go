package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestOrganizationInvalidRequestsStopBeforeService(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// A nil service proves invalid HTTP input never reaches the operation.
	h := NewOrganizationHandler(nil)
	router := gin.New()
	router.POST("/organizations", h.Store)
	router.PATCH("/organizations/:id", h.Update)
	router.POST("/organizations/:id/members", h.AddMember)

	cases := []struct {
		name   string
		method string
		path   string
		body   string
		code   string
		field  string
		rule   string
	}{
		{"missing create name", http.MethodPost, "/organizations", `{}`, "validation_failed", "name", "required"},
		{"blank create name", http.MethodPost, "/organizations", `{"name":"   "}`, "validation_failed", "name", "required"},
		{"missing update name", http.MethodPatch, "/organizations/1", `{}`, "validation_failed", "name", "required"},
		{"null update name", http.MethodPatch, "/organizations/1", `{"name":null}`, "validation_failed", "name", "required"},
		{"blank update name", http.MethodPatch, "/organizations/1", `{"name":"   "}`, "validation_failed", "name", "required"},
		{"wrong name type", http.MethodPost, "/organizations", `{"name":12}`, "validation_failed", "name", "invalid_type"},
		{"malformed request", http.MethodPost, "/organizations", `{"name":`, "malformed_json", "", ""},
		{"invalid member id", http.MethodPost, "/organizations/1/members", `{"person_id":0}`, "validation_failed", "person_id", "required"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)
			if w.Code != http.StatusBadRequest {
				t.Fatalf("status=%d want=%d body=%s", w.Code, http.StatusBadRequest, w.Body.String())
			}
			var got struct {
				Code   string                    `json:"code"`
				Errors map[string][]struct {
					Code string `json:"code"`
				} `json:"errors"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
				t.Fatal(err)
			}
			if got.Code != tt.code {
				t.Fatalf("got code=%q want=%q body=%s", got.Code, tt.code, w.Body.String())
			}
			if tt.field != "" {
				items := got.Errors[tt.field]
				if len(items) == 0 || items[0].Code != tt.rule {
					t.Fatalf("field %s: %+v want %s", tt.field, items, tt.rule)
				}
			}
		})
	}
}
