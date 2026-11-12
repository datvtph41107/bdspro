package spike

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

var routers = []struct {
	name string
	new  func() http.Handler
}{
	{"chi", Chi},
	{"gin", Gin},
	{"servemux", ServeMux},
}

func TestCommonHTTPContract(t *testing.T) {
	cases := []struct {
		name   string
		method string
		path   string
		token  bool
		body   string
		status int
		code   string
	}{
		{"health", http.MethodGet, "/api/v1/health", false, "", http.StatusOK, ""},
		{"created", http.MethodPost, "/api/v1/orgs/acme/teams", true, `{"name":"Backend"}`, http.StatusCreated, ""},
		{"auth", http.MethodPost, "/api/v1/orgs/acme/teams", false, `{"name":"Backend"}`, http.StatusUnauthorized, "unauthenticated"},
		{"invalid JSON", http.MethodPost, "/api/v1/orgs/acme/teams", true, `{"name":`, http.StatusBadRequest, "invalid_request"},
		{"unknown JSON field", http.MethodPost, "/api/v1/orgs/acme/teams", true, `{"name":"Backend","unexpected":1}`, http.StatusBadRequest, "invalid_request"},
		{"second document", http.MethodPost, "/api/v1/orgs/acme/teams", true, `{"name":"Backend"} {"name":"Another"}`, http.StatusBadRequest, "invalid_request"},
		{"invalid field", http.MethodPost, "/api/v1/orgs/acme/teams", true, `{"name":"A"}`, http.StatusUnprocessableEntity, "validation_failed"},
		{"large body", http.MethodPost, "/api/v1/orgs/acme/teams", true, strings.Repeat("x", 5000), http.StatusBadRequest, "invalid_request"},
		{"missing path", http.MethodGet, "/api/v1/missing", false, "", http.StatusNotFound, ""},
		{"wrong method", http.MethodPut, "/api/v1/health", false, "", http.StatusMethodNotAllowed, ""},
	}
	for _, router := range routers {
		t.Run(router.name, func(t *testing.T) {
			for _, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
					if tc.token {
						req.Header.Set("Authorization", "Bearer test-token")
					}
					w := httptest.NewRecorder()
					router.new().ServeHTTP(w, req)
					if w.Code != tc.status {
						t.Fatalf("status=%d want=%d body=%s", w.Code, tc.status, w.Body.String())
					}
					if w.Header().Get("X-Content-Type-Options") != "nosniff" {
						t.Fatal("missing protocol security header")
					}
					if tc.code != "" {
						var body struct { Code string `json:"code"` }
						if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil || body.Code != tc.code {
							t.Fatalf("code=%q want=%q err=%v", body.Code, tc.code, err)
						}
					}
					if tc.name == "created" {
						var body struct { Org string `json:"org"`; Name string `json:"name"` }
						if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil || body.Org != "acme" || body.Name != "Backend" {
							t.Fatalf("team=%+v err=%v", body, err)
						}
					}
				})
			}
		})
	}
}

func TestCanceledRequestContextIsPreserved(t *testing.T) {
	for _, router := range routers {
		t.Run(router.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil).WithContext(ctx)
			w := httptest.NewRecorder()
			router.new().ServeHTTP(w, req)
			if req.Context().Err() != context.Canceled {
				t.Fatal("request context cancellation lost")
			}
		})
	}
}
