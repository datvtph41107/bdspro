package context

import (
	"context"
	"net/http"
	"time"

	"github.com/datvtph41107/bdspro/modules/reqctx"
)

// RequestSource is the HTTP transport contract. A framework adapter owns
// binding, routing and response emission; Base owns request semantics.
type RequestSource interface {
	CurrentRequest() *http.Request
	PathParam(name string) string
	BindJSON(value any) error
	QueryDefault(name, fallback string) string
	WriteJSON(status int, value any)
	WriteStatus(status int)
	AbortJSON(status int, value any)
	SetHeader(name, value string)
	SetErrorCode(code string)
	IsAborted() bool
	// WrittenStatus is 0 until the HTTP response is committed.
	WrittenStatus() int
}

// Base is the single transport-independent HTTP context.
// Its lifecycle is created by the HTTP ingress, never by business services.
type Base struct {
	source RequestSource
	Data   reqctx.ContextData
}

var _ context.Context = (*Base)(nil)

func NewBaseContext(source RequestSource) *Base {
	if source == nil {
		panic("context: nil HTTP request source")
	}
	// In production APIContexter installs the datastore before creating Base.
	// Direct HTTP unit tests without middleware receive isolated local data.
	data := make(reqctx.ContextData)
	if store := reqctx.FromContext(source.CurrentRequest().Context()); store != nil {
		data = store.GetData()
	}
	return &Base{source: source, Data: data}
}

func (b *Base) GetData() reqctx.ContextData   { return b.Data }
func (b *Base) CurrentRequest() *http.Request { return b.source.CurrentRequest() }
func (b *Base) CurrentRequestContext() context.Context {
	return b.source.CurrentRequest().Context()
}
func (b *Base) Deadline() (time.Time, bool) { return b.CurrentRequestContext().Deadline() }
func (b *Base) Done() <-chan struct{}       { return b.CurrentRequestContext().Done() }
func (b *Base) Err() error                  { return b.CurrentRequestContext().Err() }
func (b *Base) Value(key any) any           { return b.CurrentRequestContext().Value(key) }

// These helpers are implemented by exactly one HTTP adapter. No extra Gin
// middleware executor, HTTP response writer or request parser is introduced.
func (b *Base) JSON(status int, value any) { b.source.WriteJSON(status, value) }
func (b *Base) Status(status int)          { b.source.WriteStatus(status) }
func (b *Base) BindJSON(value any) error   { return b.source.BindJSON(value) }
func (b *Base) DefaultQuery(name, fallback string) string {
	return b.source.QueryDefault(name, fallback)
}
func (b *Base) Header(name, value string) { b.source.SetHeader(name, value) }
func (b *Base) IsAborted() bool           { return b.source.IsAborted() }

// Written and WrittenStatus distinguish a committed response from a status
// merely selected for later emission. This is a request/transport capability,
// not a second response writer or framework-specific field on Base.
func (b *Base) Written() bool      { return b.WrittenStatus() != 0 }
func (b *Base) WrittenStatus() int { return b.source.WrittenStatus() }

func (b *Base) writeError(status int, code string, body any) {
	b.source.SetErrorCode(code)
	b.source.AbortJSON(status, body)
}
