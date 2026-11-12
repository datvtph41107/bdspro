package context

import (
	"encoding/json"
	"errors"
	"net/http"
)

// testRequestSource is a transport-only fixture; it has no domain dependencies.
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
func (s *testRequestSource) PathParam(name string) string { return s.params[name] }
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
func (s *testRequestSource) Abort() { s.aborted = true }
func (s *testRequestSource) SetHeader(name, value string) {
	if s.headers == nil {
		s.headers = make(http.Header)
	}
	s.headers.Set(name, value)
}
func (s *testRequestSource) SetErrorCode(code string) { s.code = code }
func (s *testRequestSource) IsAborted() bool { return s.aborted }
func (s *testRequestSource) WrittenStatus() int { return s.status }
