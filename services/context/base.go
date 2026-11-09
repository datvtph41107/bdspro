package context

import (
	"context"
	"net/http"
	"time"
)

// RequestSource gives Base one live view of the HTTP request and path params.
// The Gin adapter owns the concrete framework context; Base remains
// transport-neutral like Gitea's core Base contract.
type RequestSource interface {
	CurrentRequest() *http.Request
	PathParam(name string) string
}

// Base owns request-local HTTP state and context delegation. Do not construct
// it in business services; APIContexter owns its lifecycle at HTTP ingress.
type Base struct {
	source RequestSource
	Data   map[string]any
}

var _ context.Context = (*Base)(nil)

func NewBaseContext(source RequestSource) *Base {
	if source == nil {
		panic("context: nil HTTP request source")
	}
	return &Base{source: source, Data: make(map[string]any)}
}

func (b *Base) GetData() map[string]any {
	return b.Data
}

// CurrentRequestContext always follows the most recent request: authentication
// and transactional middleware can replace request.Context() during processing.
func (b *Base) CurrentRequestContext() context.Context {
	return b.source.CurrentRequest().Context()
}

func (b *Base) Deadline() (time.Time, bool) {
	return b.CurrentRequestContext().Deadline()
}

func (b *Base) Done() <-chan struct{} {
	return b.CurrentRequestContext().Done()
}

func (b *Base) Err() error {
	return b.CurrentRequestContext().Err()
}

func (b *Base) Value(key any) any {
	return b.CurrentRequestContext().Value(key)
}
