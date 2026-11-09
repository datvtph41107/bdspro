package context

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type baseRequestKey struct{}

func TestBaseUsesLatestRequestContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/orgs", nil)
	base := NewBaseContext(c)
	if got := base.Value(baseRequestKey{}); got != nil {
		t.Fatalf("unexpected initial request value: %v", got)
	}

	// Simulate middleware installing verified actor/transaction authority.
	c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), baseRequestKey{}, "verified"))
	if got := base.Value(baseRequestKey{}); got != "verified" {
		t.Fatalf("stale context propagation: got %v", got)
	}
	if base.GetData() == nil {
		t.Fatal("Base must own request-local storage")
	}
}
