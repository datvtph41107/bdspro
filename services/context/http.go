package context

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/datvtph41107/bdspro/modules/reqctx"
	"github.com/datvtph41107/bdspro/modules/web"
	"github.com/go-playground/validator/v10"
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

// trackedHTTPWriter observes real response commitment, including writes
// made by plain net/http handlers instead of Base.JSON.
type trackedHTTPWriter struct {
	http.ResponseWriter
	source *httpRequestSource
}

func (w *trackedHTTPWriter) WriteHeader(status int) {
	if w.source.committed != 0 {
		return
	}
	w.source.committed = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *trackedHTTPWriter) Write(body []byte) (int, error) {
	if w.source.committed == 0 {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(body)
}

func (w *trackedHTTPWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

// Validate JSON binding tags independently of any router framework.
var httpBinderValidator = func() *validator.Validate {
	v := validator.New()
	v.SetTagName("binding")
	return v
}()

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
	if err := json.NewDecoder(s.request.Body).Decode(value); err != nil {
		return err
	}
	return httpBinderValidator.Struct(value)
}

func (s *httpRequestSource) ClientIP() string {
	return web.ClientIPFromRequest(s.request)
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

// APIContexter creates one request-scoped HTTP APIContext; cleanup runs
// after normal completion and during panic propagation.
func APIContexter() func(http.Handler) http.Handler {
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
			source := &httpRequestSource{request: req}
			tracked := &trackedHTTPWriter{ResponseWriter: w, source: source}
			source.writer = tracked
			api := &APIContext{Base: NewBaseContext(source)}
			req = req.WithContext(context.WithValue(req.Context(), apiContextRequestKey{}, api))
			source.request = req
			defer source.finish()
			next.ServeHTTP(tracked, req)
		})
	}
}
