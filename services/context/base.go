package context

import (
	"context"
	"net/http"
	"time"
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
}

// Base is the single transport-independent HTTP context.
// Its lifecycle is created by the HTTP ingress, never by business services.
type Base struct {
	source RequestSource
	Data map[string]any
}

var _ context.Context = (*Base)(nil)

func NewBaseContext(source RequestSource) *Base {
	if source == nil {
		panic("context: nil HTTP request source")
	}
	return &Base{source: source, Data: make(map[string]any)}
}

func (b *Base) GetData() map[string]any { return b.Data }
func (b *Base) CurrentRequest() *http.Request { return b.source.CurrentRequest() }
func (b *Base) CurrentRequestContext() context.Context {
	return b.source.CurrentRequest().Context()
}
func (b *Base) Deadline() (time.Time, bool) { return b.CurrentRequestContext().Deadline() }
func (b *Base) Done() <-chan struct{} { return b.CurrentRequestContext().Done() }
func (b *Base) Err() error { return b.CurrentRequestContext().Err() }
func (b *Base) Value(key any) any { return b.CurrentRequestContext().Value(key) }

// These helpers are implemented by exactly one HTTP adapter. No extra Gin
// middleware executor, HTTP response writer or request parser is introduced.
func (b *Base) JSON(status int, value any) { b.source.WriteJSON(status, value) }
func (b *Base) Status(status int) { b.source.WriteStatus(status) }
func (b *Base) BindJSON(value any) error { return b.source.BindJSON(value) }
func (b *Base) DefaultQuery(name, fallback string) string { return b.source.QueryDefault(name, fallback) }
func (b *Base) Header(name, value string) { b.source.SetHeader(name, value) }
func (b *Base) IsAborted() bool { return b.source.IsAborted() }

func (b *Base) writeError(status int, code string, body any) {
	b.source.SetErrorCode(code)
	b.source.AbortJSON(status, body)
}
