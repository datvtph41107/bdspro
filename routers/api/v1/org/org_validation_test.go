package org

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	appctx "github.com/datvtph41107/bdspro/services/context"
	"github.com/gin-gonic/gin"
)

func TestOrganizationInvalidRequestsStopBeforeService(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// A nil service proves invalid HTTP input never reaches the operation.
	h := NewHandler(nil)
	router := gin.New()
	router.POST("/orgs", appctx.Wrap(h.Create))
	router.PATCH("/orgs/:id", appctx.Wrap(h.Edit))
	router.POST("/orgs/:id/members", appctx.Wrap(h.AddMember))

	cases := []struct {
		name   string
		method string
		path   string
		body   string
		code   string
		field  string
		rule   string
	}{
		{"missing create name", http.MethodPost, "/orgs", `{}`, "validation_failed", "name", "required"},
		{"blank create name", http.MethodPost, "/orgs", `{"name":"   "}`, "validation_failed", "name", "required"},
		{"missing update name", http.MethodPatch, "/orgs/1", `{}`, "validation_failed", "name", "required"},
		{"null update name", http.MethodPatch, "/orgs/1", `{"name":null}`, "validation_failed", "name", "required"},
		{"blank update name", http.MethodPatch, "/orgs/1", `{"name":"   "}`, "validation_failed", "name", "required"},
		{"wrong name type", http.MethodPost, "/orgs", `{"name":12}`, "validation_failed", "name", "invalid_type"},
		{"malformed request", http.MethodPost, "/orgs", `{"name":`, "malformed_json", "", ""},
		{"invalid member id", http.MethodPost, "/orgs/1/members", `{"person_id":0}`, "validation_failed", "person_id", "required"},
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
				Code   string `json:"code"`
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
