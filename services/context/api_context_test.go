package context

import (
	"context"
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

type dynamicRequestKey struct{}

func TestGinAdapterCarriesPathAndUpdatedContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(APIContexter())
	router.GET("/orgs/:id", func(c *gin.Context) {
		api := GetAPIContext(c)
		id, err := api.PositivePathID("id")
		if err != nil || id != 42 {
			t.Fatalf("path ID=%d err=%v", id, err)
		}
		c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), dynamicRequestKey{}, "live"))
		if got := api.Value(dynamicRequestKey{}); got != "live" {
			t.Fatalf("APIContext kept a stale request: got=%v", got)
		}
		if api.Context != c {
			t.Fatal("Gin compatibility bridge did not retain one request owner")
		}
		api.Status(http.StatusNoContent)
	})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/orgs/42", nil))
	if w.Code != http.StatusNoContent {
		t.Fatalf("status=%d", w.Code)
	}
}
