package context

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLinkHeaderUsesValidatedPage(t *testing.T) {
	// The header writer must use the caller's validated options, not reparse
	// raw query/form data or silently choose a different page.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/orgs/1/teams?page=99&limit=10", nil)
	w := httptest.NewRecorder()
	APIContexter()(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		ctx := GetAPIContextFromRequest(req)
		ctx.SetLinkHeader(41, 2, 10)
		ctx.SetTotalCountHeader(41)
		ctx.Status(http.StatusOK)
	})).ServeHTTP(w, req)

	link := w.Header().Get("Link")
	for _, expected := range []string{
		"page=3>; rel=\"next\"",
		"page=5>; rel=\"last\"",
		"page=1>; rel=\"first\"",
		"page=1>; rel=\"prev\"",
	} {
		if !strings.Contains(link, expected) {
			t.Fatalf("missing %q in Link: %s", expected, link)
		}
	}
	if w.Header().Get("X-Total-Count") != "41" {
		t.Fatal("wrong total count")
	}
	if strings.Contains(link, "page=99") {
		t.Fatal("raw page leaked into canonical links")
	}
}
