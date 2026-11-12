package context

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/datvtph41107/bdspro/modules/reqctx"
)

func TestJSONCannotAppendAfterCommittedResponse(t *testing.T) {
	request, done := reqctx.NewRequestContext(context.Background())
	defer done()
	w := httptest.NewRecorder()
	ctx := NewBaseContext(w, httptest.NewRequest(http.MethodGet, "/", nil).WithContext(request))
	ctx.JSON(http.StatusCreated, map[string]string{"state": "created"})
	before := w.Body.String()
	ctx.JSON(http.StatusInternalServerError, map[string]string{"error": "should not be appended"})
	if w.Code != http.StatusCreated || w.Body.String() != before {
		t.Fatalf("response changed after commit: status=%d body=%q", w.Code, w.Body.String())
	}
}
