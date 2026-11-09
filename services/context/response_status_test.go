package context

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// The public response contract is committed HTTP status, not Gin's default
// status or a chosen-but-unwritten status.
func TestBaseWrittenStatusMatchesGinResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(APIContexter())
	router.GET("/status", func(c *gin.Context) {
		api := GetAPIContext(c)
		if api.Written() || api.WrittenStatus() != 0 {
			t.Fatalf("new response reported as committed: %d", api.WrittenStatus())
		}
		api.Status(http.StatusAccepted)
		if api.Written() || api.WrittenStatus() != 0 {
			t.Fatalf("unflushed response status reported as committed: %d", api.WrittenStatus())
		}
		api.JSON(http.StatusCreated, map[string]bool{"created": true})
		if !api.Written() || api.WrittenStatus() != http.StatusCreated {
			t.Fatalf("committed response state=%v status=%d", api.Written(), api.WrittenStatus())
		}
	})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/status", nil))
	if w.Code != http.StatusCreated {
		t.Fatalf("response status=%d", w.Code)
	}
}
