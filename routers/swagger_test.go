package routers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	api "github.com/datvtph41107/bdspro/routers/api"
	"github.com/gin-gonic/gin"
)

func TestSwaggerAssetsDevelopmentOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cases := []struct {
		mode     string
		path     string
		status   int
		fragment string
	}{
		{"development", "/swagger/openapi.json", http.StatusOK, `"openapi": "3.0.3"`},
		{"development", "/swagger/", http.StatusOK, "Swagger UI"},
		{"production", "/swagger/openapi.json", http.StatusNotFound, ""},
	}
	for _, tc := range cases {
		t.Run(tc.mode+tc.path, func(t *testing.T) {
			r := gin.New()
			api.RegisterSwagger(r, tc.mode)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, tc.path, nil))
			if w.Code != tc.status {
				t.Fatalf("status %d want %d", w.Code, tc.status)
			}
			if tc.fragment != "" && !strings.Contains(w.Body.String(), tc.fragment) {
				t.Fatalf("asset %s does not contain %q", tc.path, tc.fragment)
			}
		})
	}
}
