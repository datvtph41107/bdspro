package context

import (
	"bufio"
	"net"
	"net/http"

	web_types "github.com/datvtph41107/bdspro/modules/web/types"
)

type ResponseWriter interface {
	http.ResponseWriter
	http.Flusher
	web_types.ResponseStatusProvider
	Before(func(ResponseWriter))
	WrittenSize() int
}

type Response struct {
	http.ResponseWriter
	status int
	written int
	before []func(ResponseWriter)
	beforeExecuted bool
}

var _ ResponseWriter = (*Response)(nil)

func WrapResponseWriter(w http.ResponseWriter) *Response {
	if resp, ok := w.(*Response); ok {
		return resp
	}
	if w == nil {
		panic("context: nil response writer")
	}
	return &Response{ResponseWriter: w}
}

func (r *Response) runBefore() {
	if r.beforeExecuted {
		return
	}
	r.beforeExecuted = true
	for _, fn := range r.before {
		fn(r)
	}
}

func (r *Response) Before(fn func(ResponseWriter)) {
	if fn == nil {
		return
	}
	if r.beforeExecuted {
		panic("context: response already committed")
	}
	r.before = append(r.before, fn)
}

func (r *Response) WriteHeader(status int) {
	if r.status != 0 {
		return
	}
	r.runBefore()
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *Response) Write(data []byte) (int, error) {
	if r.status == 0 {
		r.WriteHeader(http.StatusOK)
	}
	n, err := r.ResponseWriter.Write(data)
	r.written += n
	return n, err
}

func (r *Response) WrittenStatus() int { return r.status }
func (r *Response) WrittenSize() int { return r.written }

func (r *Response) Flush() {
	if r.status == 0 {
		r.WriteHeader(http.StatusOK)
	}
	if f, ok := r.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (r *Response) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := r.ResponseWriter.(http.Hijacker); ok {
		return h.Hijack()
	}
	return nil, nil, http.ErrNotSupported
}

func (r *Response) Unwrap() http.ResponseWriter { return r.ResponseWriter }
