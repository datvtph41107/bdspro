package context

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/datvtph41107/bdspro/modules/web"
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

// End-to-end typed action execution must consult APIContext.WrittenStatus,
// while retaining the selected-but-unwritten status semantics.
func TestTypedAPIContextResponseStatusControlsActionChain(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(APIContexter())
	m := web.NewRouter[*APIContext](router.Group(""), GetAPIContext)

	reachedAfterWritten := false
	m.Get("/written", func(c *APIContext) {
		c.JSON(http.StatusAccepted, map[string]bool{"ok": true})
	}, func(*APIContext) { reachedAfterWritten = true })
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/written", nil))
	if reachedAfterWritten || w.Code != http.StatusAccepted {
		t.Fatalf("committed response did not stop action chain: reached=%v status=%d", reachedAfterWritten, w.Code)
	}

	reachedAfterStatusOnly := false
	m.Get("/status-only", func(c *APIContext) {
		c.Status(http.StatusAccepted)
	}, func(c *APIContext) {
		reachedAfterStatusOnly = true
		c.JSON(http.StatusOK, map[string]bool{"ok": true})
	})
	w = httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/status-only", nil))
	if !reachedAfterStatusOnly || w.Code != http.StatusOK {
		t.Fatalf("status-only prematurely aborted: reached=%v status=%d", reachedAfterStatusOnly, w.Code)
	}
}
