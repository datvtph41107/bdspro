package context

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/datvtph41107/bdspro/modules/reqctx"
	"github.com/gin-gonic/gin"
)

type requestStoreTraceKey struct{}

func TestAPIContexterUsesModulesRequestDataAndCleanup(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(APIContexter())
	cleanupCount := 0
	router.GET("/orgs/:id", func(c *gin.Context) {
		store := reqctx.FromContext(c.Request.Context())
		if store == nil {
			t.Fatal("request scope was not initialized before route action")
		}
		store.AddCleanup(func() { cleanupCount++ })
		api := GetAPIContext(c)
		api.GetData()["form"] = "bound"
		if store.GetData()["form"] != "bound" {
			t.Fatal("Base has a second datastore")
		}
		c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), requestStoreTraceKey{}, "current"))
		if api.Value(requestStoreTraceKey{}) != "current" || reqctx.FromContext(c.Request.Context()) != store {
			t.Fatal("middleware lost request context or datastore")
		}
		api.Status(http.StatusNoContent)
	})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/orgs/42", nil))
	if response.Code != http.StatusNoContent || cleanupCount != 1 {
		t.Fatalf("status=%d cleanup=%d", response.Code, cleanupCount)
	}
}
