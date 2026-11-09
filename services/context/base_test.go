package context

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

type baseRequestKey struct{}

type testRequestSource struct {
	req    *http.Request
	params map[string]string
}

func (s *testRequestSource) CurrentRequest() *http.Request { return s.req }
func (s *testRequestSource) PathParam(name string) string  { return s.params[name] }

func TestBaseUsesLatestRequestContext(t *testing.T) {
	source := &testRequestSource{
		req:    httptest.NewRequest(http.MethodGet, "/orgs", nil),
		params: make(map[string]string),
	}
	base := NewBaseContext(source)
	if got := base.Value(baseRequestKey{}); got != nil {
		t.Fatalf("unexpected initial request value: %v", got)
	}

	// A verified actor and transaction can be installed after Base is created.
	source.req = source.req.WithContext(context.WithValue(source.req.Context(), baseRequestKey{}, "verified"))
	if got := base.Value(baseRequestKey{}); got != "verified" {
		t.Fatalf("stale context propagation: got %v", got)
	}
	if base.GetData() == nil {
		t.Fatal("Base must own request-local storage")
	}
}
