package context

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/datvtph41107/bdspro/modules/reqctx"
)

func TestHTTPNativeAPIContexterRequestAndCleanup(t *testing.T) {
	type actorKey struct{}
	cleanups := 0
	mux := http.NewServeMux()
	mux.Handle("POST /orgs/{org}/teams/{id}", APIContexterHTTP()(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		api := GetAPIContextFromRequest(req)
		if api.PathParam("org") != "acme" {
			t.Errorf("org=%q", api.PathParam("org"))
		}
		if id, err := api.PositivePathID("id"); id != 42 || err != nil {
			t.Errorf("id=%d err=%v", id, err)
		}
		if got := api.ClientIP(); got != "127.0.0.1" {
			t.Errorf("client IP=%q", got)
		}
		if api.DefaultQuery("mode", "standard") != "test" {
			t.Error("query lost")
		}
		store := reqctx.FromContext(req.Context())
		if store == nil || store.GetData() != nil && api.GetData() == nil {
			t.Fatal("request datastore missing")
		}
		store.AddCleanup(func() { cleanups++ })
		var form struct {
			Name string `json:"name"`
		}
		if err := api.BindJSON(&form); err != nil || form.Name != "backend" {
			t.Fatalf("BindJSON form=%+v err=%v", form, err)
		}
		api.JSON(http.StatusCreated, map[string]string{"team": form.Name})
		if !api.Written() || api.WrittenStatus() != http.StatusCreated {
			t.Error("HTTP response not committed")
		}
	})))
	withActor := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			next.ServeHTTP(w, req.WithContext(context.WithValue(req.Context(), actorKey{}, 17)))
		})
	}
	req := httptest.NewRequest(http.MethodPost, "/orgs/acme/teams/42?mode=test", strings.NewReader(`{"name":"backend"}`))
	req.RemoteAddr = "127.0.0.1:1234"
	rec := httptest.NewRecorder()
	withActor(mux).ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated || cleanups != 1 {
		t.Fatalf("status=%d cleanups=%d body=%q", rec.Code, cleanups, rec.Body.String())
	}
	var data map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &data); err != nil || data["team"] != "backend" {
		t.Errorf("JSON data=%v err=%v", data, err)
	}
}

func TestHTTPNativeAPIContexterSelectionAndAbort(t *testing.T) {
	nextCalled := false
	handler := APIContexterHTTP()(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		api := GetAPIContextFromRequest(req)
		api.Status(http.StatusAccepted)
		if api.Written() {
			t.Fatal("selecting status must not commit")
		}
		api.Abort()
		if !api.IsAborted() {
			t.Fatal("abort state lost")
		}
		nextCalled = true
		api.Status(http.StatusNoContent)
	}))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusNoContent || !nextCalled {
		t.Errorf("status=%d called=%t", rec.Code, nextCalled)
	}
}

func TestHTTPNativeAPIContexterPublicError(t *testing.T) {
	handler := APIContexterHTTP()(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		api := GetAPIContextFromRequest(req)
		api.AuthenticationRequired()
		if api.WrittenStatus() != http.StatusUnauthorized {
			t.Errorf("committed=%d", api.WrittenStatus())
		}
	}))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusUnauthorized || rec.Header().Get("WWW-Authenticate") != "Bearer" {
		t.Errorf("status=%d authenticate=%q", rec.Code, rec.Header().Get("WWW-Authenticate"))
	}
}
