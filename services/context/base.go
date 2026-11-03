package context

import (
	"context"
	"net/http"
	"time"
)

// Base holds the state of a single HTTP request. Business services accept
// context.Context, never *Base or *APIContext.
type Base struct {
	Req  *http.Request
	Resp *Response
}

func NewBaseContext(w http.ResponseWriter, r *http.Request) *Base {
	if r == nil {
		panic("context: nil request")
	}
	return &Base{Req: r, Resp: WrapResponseWriter(w)}
}

func (b *Base) Deadline() (deadline time.Time, ok bool) { return b.Req.Context().Deadline() }
func (b *Base) Done() <-chan struct{}                   { return b.Req.Context().Done() }
func (b *Base) Err() error                              { return b.Req.Context().Err() }
func (b *Base) Value(key any) any                       { return b.Req.Context().Value(key) }

var _ context.Context = (*Base)(nil)
