package context

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type baseRequestKey struct{}

type testRequestSource struct {
	req     *http.Request
	params  map[string]string
	status  int
	body    []byte
	code    string
	aborted bool
	headers http.Header
}

func (s *testRequestSource) CurrentRequest() *http.Request { return s.req }
func (s *testRequestSource) PathParam(name string) string  { return s.params[name] }
func (s *testRequestSource) BindJSON(value any) error {
	if s.req.Body == nil {
		return errors.New("empty body")
	}
	return json.NewDecoder(s.req.Body).Decode(value)
}
func (s *testRequestSource) QueryDefault(name, fallback string) string {
	if value, exists := s.req.URL.Query()[name]; exists && len(value) > 0 {
		return value[0]
	}
	return fallback
}
func (s *testRequestSource) WriteJSON(status int, encoded []byte) {
	s.status, s.body = status, append([]byte(nil), encoded...)
}
func (s *testRequestSource) WriteStatus(status int) { s.status = status }
func (s *testRequestSource) Abort()                 { s.aborted = true }
func (s *testRequestSource) SetHeader(name, value string) {
	if s.headers == nil {
		s.headers = make(http.Header)
	}
	s.headers.Set(name, value)
}
func (s *testRequestSource) SetErrorCode(code string) { s.code = code }
func (s *testRequestSource) IsAborted() bool          { return s.aborted }
func (s *testRequestSource) WrittenStatus() int       { return s.status }

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

func TestAPIContextCanWriteWithoutGin(t *testing.T) {
	source := &testRequestSource{
		req:    httptest.NewRequest(http.MethodPost, "/orgs?limit=15", nil),
		params: map[string]string{"id": "42"},
	}
	api := &APIContext{Base: NewBaseContext(source), Org: &APIOrganization{}}
	if got := api.DefaultQuery("limit", "20"); got != "15" {
		t.Fatalf("query=%q", got)
	}
	if id, err := api.PositivePathID("id"); err != nil || id != 42 {
		t.Fatalf("ID=%d err=%v", id, err)
	}
	api.JSON(http.StatusCreated, map[string]any{"id": 42})
	if source.status != http.StatusCreated || source.aborted {
		t.Fatalf("unexpected normal response: status=%d abort=%v", source.status, source.aborted)
	}
	api.APIError(http.StatusForbidden, "owner_required", "owner required")
	if source.status != http.StatusForbidden || !source.aborted || source.code != "owner_required" {
		t.Fatalf("unexpected error response: status=%d abort=%v code=%q", source.status, source.aborted, source.code)
	}
	var body APIErrorResponse
	if err := json.Unmarshal(source.body, &body); err != nil || body.Code != "owner_required" || body.Error != "owner required" {
		t.Fatalf("error contract changed: %s, err=%v", source.body, err)
	}
}

func TestAPIContextValidationWithoutGin(t *testing.T) {
	source := &testRequestSource{req: httptest.NewRequest(http.MethodPost, "/orgs", nil)}
	api := &APIContext{Base: NewBaseContext(source), Org: &APIOrganization{}}
	api.Validation(errors.New("not JSON"), &struct {
		Name string `json:"name"`
	}{})
	if source.status != http.StatusBadRequest || source.code != "invalid_request_body" || !source.aborted {
		t.Fatalf("invalid request: status=%d code=%q aborted=%v", source.status, source.code, source.aborted)
	}
}

func TestBaseJSONUsesRelaxedCodecAndNeverExposesMarshalErrors(t *testing.T) {
	source := &testRequestSource{req: httptest.NewRequest(http.MethodGet, "/test", nil)}
	api := &APIContext{Base: NewBaseContext(source)}

	api.JSON(http.StatusOK, map[string]string{"value": string([]byte{0xff})})
	var decoded map[string]string
	if err := json.Unmarshal(source.body, &decoded); err != nil || decoded["value"] != "\ufffd" {
		t.Fatalf("relaxed JSON response=%s err=%v", source.body, err)
	}

	api.JSON(http.StatusOK, map[string]any{"not_json": make(chan int)})
	var public APIErrorResponse
	if err := json.Unmarshal(source.body, &public); err != nil || source.status != http.StatusInternalServerError || source.code != "internal_error" || public.Code != "internal_error" || public.Error != "internal server error" {
		t.Fatalf("invalid JSON escaped response contract: status=%d code=%q body=%s err=%v", source.status, source.code, source.body, err)
	}
}
