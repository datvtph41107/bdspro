package routers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func TestNormalRoutesRequestIDBeforeMatching(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NormalRoutes().WrapIngress()
	for _, tc := range []struct{ method, path string }{
		{http.MethodGet, "/health"},
		{http.MethodGet, "/__missing_path"},
		{http.MethodHead, "/__missing_path"},
	} {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, nil))
			id := w.Header().Get("X-Request-ID")
			if id == "" {
				t.Fatalf("HTTP %d returned no request ID", w.Code)
			}
			if _, err := uuid.Parse(id); err != nil {
				t.Fatalf("invalid request ID %q: %v", id, err)
			}
		})
	}
}
