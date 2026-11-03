package context

import (
	"bufio"
	"net"
	"net/http"
)

// Response records HTTP status and bytes while preserving access to the
// underlying writer for standard ResponseController operations.
// A response is request-confined; it must not be used by parallel goroutines.
type Response struct {
	http.ResponseWriter
	status int
	size   int
}

func WrapResponseWriter(w http.ResponseWriter) *Response {
	if w == nil {
		panic("context: nil ResponseWriter")
	}
	if r, ok := w.(*Response); ok {
		return r
	}
	return &Response{ResponseWriter: w}
}

// Unwrap allows http.NewResponseController to reach capabilities of the actual writer.
func (r *Response) Unwrap() http.ResponseWriter { return r.ResponseWriter }

func (r *Response) WriteHeader(status int) {
	if r.status != 0 {
		return
	}
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *Response) Write(p []byte) (int, error) {
	if r.status == 0 {
		r.WriteHeader(http.StatusOK)
	}
	n, err := r.ResponseWriter.Write(p)
	r.size += n
	return n, err
}

func (r *Response) WrittenStatus() int { return r.status }
func (r *Response) WrittenSize() int   { return r.size }

// Flush is useful to existing call sites. New code can use ResponseController.
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
