package context

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBeforeHeaderCallbackCommitsOnlyOnce(t *testing.T) {
	w := httptest.NewRecorder()
	resp := WrapResponseWriter(w)
	calls := 0
	resp.Before(func(writer ResponseWriter) {
		calls++
		writer.WriteHeader(http.StatusAccepted)
	})
	resp.WriteHeader(http.StatusBadRequest)
	if calls != 1 || resp.WrittenStatus() != http.StatusAccepted || w.Code != http.StatusAccepted {
		t.Fatalf("calls=%d recorded=%d wire=%d", calls, resp.WrittenStatus(), w.Code)
	}
}

func TestResponseTracksImplicitStatusAndBody(t *testing.T) {
	w := httptest.NewRecorder()
	resp := WrapResponseWriter(w)
	n, err := resp.Write([]byte("abc"))
	if err != nil || n != 3 || resp.WrittenStatus() != http.StatusOK || resp.WrittenSize() != 3 {
		t.Fatalf("n=%d err=%v status=%d size=%d", n, err, resp.WrittenStatus(), resp.WrittenSize())
	}
}
