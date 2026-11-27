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

type RequestSource interface {
	CurrentRequest() *http.Request
	BindJSON(value any) error

	WriteJSON(status int, encoded []byte)
	WriteStatus(status int)
	Abort()
	SetHeader(name, value string)
	ResponseHeader() http.Header
	SetErrorCode(code string)
	IsAborted() bool

	WrittenStatus() int
}

type Base struct {
	source   RequestSource
	lifetime context.Context
	Data     reqctx.ContextData
}

var _ context.Context = (*Base)(nil)
var _ web_types.ResponseStatusProvider = (*Base)(nil)

func NewBaseContext(source RequestSource) *Base {
	if source == nil {
		panic("context: nil HTTP request source")
	}

	lifetime := source.CurrentRequest().Context()
	data := make(reqctx.ContextData)
	if store := reqctx.FromContext(lifetime); store != nil {
		data = store.GetData()
	}
	return &Base{source: source, lifetime: lifetime, Data: data}
}

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

func (b *Base) Deadline() (time.Time, bool) { return b.lifetime.Deadline() }
func (b *Base) Done() <-chan struct{}       { return b.lifetime.Done() }
func (b *Base) Err() error                  { return b.lifetime.Err() }

func (b *Base) Value(key any) any {
	if value := b.lifetime.Value(key); value != nil {
		return value
	}
	return b.CurrentRequestContext().Value(key)
}

func (b *Base) JSON(status int, value any) {

	if b.Written() {
		return
	}
	var encoded bytes.Buffer
	if err := appjson.MarshalWriteRelaxed(&encoded, value); err != nil {

		log.Ctx(b).Error().Str("error_type", fmt.Sprintf("%T", err)).Msg("http.json_encode_error")
		b.source.SetErrorCode("internal_error")
		b.source.WriteJSON(http.StatusInternalServerError, []byte(`{"error":"internal server error","code":"internal_error"}`))
		return
	}
	b.source.WriteJSON(status, encoded.Bytes())
}
func (b *Base) Status(status int) {
	if b.Written() {
		return
	}
	b.source.WriteStatus(status)
}
func (b *Base) BindJSON(value any) error { return b.source.BindJSON(value) }
func (b *Base) Header(name, value string) { b.source.SetHeader(name, value) }
func (b *Base) IsAborted() bool           { return b.source.IsAborted() }

func (b *Base) Abort() { b.source.Abort() }

func (b *Base) Written() bool      { return b.WrittenStatus() != 0 }
func (b *Base) WrittenStatus() int { return b.source.WrittenStatus() }

func (b *Base) writeError(status int, code string, body any) {

	if b.Written() {
		b.source.Abort()
		return
	}
	b.source.SetErrorCode(code)
	b.JSON(status, body)
	b.source.Abort()
}
