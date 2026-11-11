package common

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

func TestRequestIDHeaderAndContext(t *testing.T) {
	var innerID string
	handler := RequestID()(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		innerID = w.Header().Get("X-Request-ID")
		w.WriteHeader(http.StatusNoContent)
	}))

	for i := 0; i < 2; i++ {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/check", nil))
		id := rec.Header().Get("X-Request-ID")
		if rec.Code != http.StatusNoContent || id == "" || id != innerID {
			t.Fatalf("status=%d header=%q forwarded=%q", rec.Code, id, innerID)
		}
		if _, err := uuid.Parse(id); err != nil {
			t.Fatalf("invalid request ID %q: %v", id, err)
		}
	}
}
