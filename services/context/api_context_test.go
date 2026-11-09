package context

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAPIContextIsSharedWithinRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(APIContexter())
	var first *APIContext
	r.GET("/one", func(c *gin.Context) {
		first = GetAPIContext(c)
		first.GetData()["check"] = true
	}, Wrap(func(api *APIContext) {
		if api != first || api.GetData()["check"] != true {
			t.Fatal("APIContext was rebuilt between handlers")
		}
		api.Status(http.StatusNoContent)
	}))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/one", nil))
	if w.Code != http.StatusNoContent {
		t.Fatalf("status=%d", w.Code)
	}
}
