package context

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"time"

	appjson "github.com/datvtph41107/bdspro/modules/json"
	"github.com/datvtph41107/bdspro/modules/log"
	"github.com/datvtph41107/bdspro/modules/reqctx"
	web_types "github.com/datvtph41107/bdspro/modules/web/types"
)

// RequestSource is the HTTP transport contract. A framework adapter owns
// binding, routing and response emission; Base owns request semantics.
type RequestSource interface {
	CurrentRequest() *http.Request
	PathParam(name string) string
	BindJSON(value any) error
	QueryDefault(name, fallback string) string
	// WriteJSON writes bytes already encoded by Base.JSON, never re-marshals.
	WriteJSON(status int, encoded []byte)
	WriteStatus(status int)
	Abort()
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
var _ web_types.ResponseStatusProvider = (*Base)(nil)

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

// ClientIP is assigned by the trusted HTTP transport, not derived from an
// unvalidated request header. An unsupported transport fails closed.
func (b *Base) ClientIP() string {
	source, ok := b.source.(interface{ ClientIP() string })
	if !ok {
		panic("context: request transport has no trusted client IP")
	}
	return source.ClientIP()
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
// JSON is the canonical JSON response writer, matching Gitea's Base.JSON
// -> modules/json.MarshalWriteRelaxed boundary. The Gin adapter emits bytes
// only; it must never choose another JSON codec.
func (b *Base) JSON(status int, value any) {
	var encoded bytes.Buffer
	if err := appjson.MarshalWriteRelaxed(&encoded, value); err != nil {
		// Never leak provider or driver data through a failed response renderer.
		log.Ctx(b).Error().Str("error_type", fmt.Sprintf("%T", err)).Msg("http.json_encode_error")
		b.source.SetErrorCode("internal_error")
		b.source.WriteJSON(http.StatusInternalServerError, []byte(`{"error":"internal server error","code":"internal_error"}`))
		return
	}
	b.source.WriteJSON(status, encoded.Bytes())
}
func (b *Base) Status(status int)        { b.source.WriteStatus(status) }
func (b *Base) BindJSON(value any) error { return b.source.BindJSON(value) }
func (b *Base) DefaultQuery(name, fallback string) string {
	return b.source.QueryDefault(name, fallback)
}
func (b *Base) Header(name, value string) { b.source.SetHeader(name, value) }
func (b *Base) IsAborted() bool           { return b.source.IsAborted() }

// Abort ends the typed matched-route chain without committing a response.
// This is a transport control, not a second HTTP response writer.
func (b *Base) Abort() { b.source.Abort() }

// Written and WrittenStatus distinguish a committed response from a status
// merely selected for later emission. This is a request/transport capability,
// not a second response writer or framework-specific field on Base.
func (b *Base) Written() bool      { return b.WrittenStatus() != 0 }
func (b *Base) WrittenStatus() int { return b.source.WrittenStatus() }

func (b *Base) writeError(status int, code string, body any) {
	b.source.SetErrorCode(code)
	b.JSON(status, body)
	b.source.Abort()
}
