package context

import (
	"context"
	"encoding/json"
	"github.com/datvtph41107/bdspro/modules/reqctx"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestResponseLifecycleAndJSON(t *testing.T) {
	w := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/users/me", nil)
	reqctx.RequestContextHandler()(APIContexter()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := GetAPIContext(r)
		if ctx.Req != r {
			t.Fatal("request mismatch")
		}
		ctx.Doer = &Principal{AccountID: 9}
		if ctx.Doer.AccountID != 9 {
			t.Fatal("principal missing")
		}
		if ctx.WrittenStatus() != 0 {
			t.Fatal("unexpected response before handler")
		}
		ctx.JSON(http.StatusCreated, map[string]string{"ok": "true"})
		if ctx.WrittenStatus() != http.StatusCreated {
			t.Fatal("written status mismatch")
		}
		ctx.APIError(http.StatusForbidden, "ignored", "ignored")
	}))).ServeHTTP(w, request)
	if w.Code != http.StatusCreated {
		t.Fatalf("status %d", w.Code)
	}
	if got := w.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("content type %q", got)
	}
	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(body, map[string]string{"ok": "true"}) {
		t.Fatalf("body %v", body)
	}
}

func TestAPIErrorSanitizesInternals(t *testing.T) {
	w := httptest.NewRecorder()
	reqctx.RequestContextHandler()(APIContexter()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := GetAPIContext(r)
		ctx.APIError(http.StatusInternalServerError, "db.error", "password=secret")
	}))).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	if w.Code != 500 {
		t.Fatalf("status %d", w.Code)
	}
	if got := w.Body.String(); got != "{\"error\":\"internal server error\",\"code\":\"internal_error\"}" {
		t.Errorf("unsafe error response: %s", got)
	}
}

func TestCancellationAndRequestIsolation(t *testing.T) {
	parent, cancel := context.WithCancel(context.Background())
	defer cancel()
	first := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(parent)
	var cached *APIContext
	handler := reqctx.RequestContextHandler()(APIContexter()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c := GetAPIContext(r)
		if cached != nil && c == cached {
			t.Error("reused APIContext")
		}
		cached = c
	})))
	handler.ServeHTTP(httptest.NewRecorder(), first)
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	cancel()
	if cached == nil {
		t.Fatal("no APIContext")
	}
}

func TestResponseWrittenSizeAndImplicit200(t *testing.T) {
	w := httptest.NewRecorder()
	r := WrapResponseWriter(w)
	n, err := r.Write([]byte("hello"))
	if err != nil || n != 5 || r.WrittenStatus() != 200 || r.WrittenSize() != 5 {
		t.Fatalf("%d %v %d %d", n, err, r.WrittenStatus(), r.WrittenSize())
	}
	r.WriteHeader(404)
	if r.WrittenStatus() != 200 {
		t.Fatal("double WriteHeader changed status")
	}
}

func BenchmarkAPIContexter(b *testing.B) {
	h := reqctx.RequestContextHandler()(APIContexter()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		GetAPIContext(r).Resp.WriteHeader(http.StatusNoContent)
	})))
	r := httptest.NewRequest(http.MethodGet, "/health", nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.ServeHTTP(httptest.NewRecorder(), r)
	}
}
