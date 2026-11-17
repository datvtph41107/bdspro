package context

import (
	"context"
	"encoding/json"
	"net"
	"net/http"

	"github.com/datvtph41107/bdspro/modules/reqctx"
)

// httpRequestSource owns request/response state for an HTTP-native
// APIContexter. It is request-local and does not depend on a route matcher.
// The selected HTTP status is not committed until WriteJSON or finish.
type httpRequestSource struct {
	request   *http.Request
	writer    http.ResponseWriter
	selected  int
	committed int
	aborted   bool
}

var _ RequestSource = (*httpRequestSource)(nil)

func (s *httpRequestSource) CurrentRequest() *http.Request { return s.request }

func (s *httpRequestSource) PathParam(name string) string {
	return s.request.PathValue(name)
}

func (s *httpRequestSource) QueryDefault(name, fallback string) string {
	if values, ok := s.request.URL.Query()[name]; ok && len(values) > 0 {
		return values[0]
	}
	return fallback
}

func (s *httpRequestSource) BindJSON(value any) error {
	return json.NewDecoder(s.request.Body).Decode(value)
}

func (s *httpRequestSource) ClientIP() string {
	host, _, err := net.SplitHostPort(s.request.RemoteAddr)
	if err != nil {
		return s.request.RemoteAddr
	}
	// Never trust X-Forwarded-For without an explicit trusted-proxy policy.
	return host
}

func (s *httpRequestSource) WriteJSON(status int, encoded []byte) {
	if s.committed != 0 {
		return
	}
	s.writer.Header().Set("Content-Type", "application/json;charset=utf-8")
	s.writeHeader(status)
	_, _ = s.writer.Write(encoded)
}

func (s *httpRequestSource) writeHeader(status int) {
	if s.committed != 0 {
		return
	}
	s.committed = status
	s.writer.WriteHeader(status)
}

func (s *httpRequestSource) WriteStatus(status int) {
	if s.committed == 0 {
		s.selected = status
	}
}
func (s *httpRequestSource) Abort()                { s.aborted = true }
func (s *httpRequestSource) IsAborted() bool       { return s.aborted }
func (s *httpRequestSource) WrittenStatus() int    { return s.committed }
func (s *httpRequestSource) SetHeader(k, v string) { s.writer.Header().Set(k, v) }
func (s *httpRequestSource) SetErrorCode(code string) {
	if store := reqctx.FromContext(s.request.Context()); store != nil {
		store.GetData()[ErrorCodeKey] = code
	}
}

func (s *httpRequestSource) finish() {
	if s.committed == 0 && s.selected != 0 {
		s.writeHeader(s.selected)
	}
}

// APIContexterHTTP is the native HTTP sibling of the legacy Gin-backed
// APIContexter. It will become the canonical APIContexter when the router
// matcher is migrated. Request data cleanup always runs, including on panic.
func APIContexterHTTP() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if existing, ok := req.Context().Value(apiContextRequestKey{}).(*APIContext); ok && existing != nil {
				next.ServeHTTP(w, req)
				return
			}
			if reqctx.FromContext(req.Context()) == nil {
				scoped, finished := reqctx.NewRequestContext(req.Context())
				defer finished()
				req = req.WithContext(scoped)
			}
			source := &httpRequestSource{request: req, writer: w}
			api := &APIContext{Base: NewBaseContext(source)}
			req = req.WithContext(context.WithValue(req.Context(), apiContextRequestKey{}, api))
			source.request = req
			defer source.finish()
			next.ServeHTTP(w, req)
		})
	}
}
