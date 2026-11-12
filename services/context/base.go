package context

import (
	"bytes"
	"context"
	"net/http"

	appjson "github.com/datvtph41107/bdspro/modules/json"
	"github.com/datvtph41107/bdspro/modules/reqctx"
	web_types "github.com/datvtph41107/bdspro/modules/web/types"
)

type BaseContextKeyType struct{}

var BaseContextKey = BaseContextKeyType{}

// Base is created by HTTP middleware, never by domain services.
type Base struct {
	reqctx.RequestContext
	Resp ResponseWriter
	Req  *http.Request
	Data reqctx.ContextData
}

var _ context.Context = (*Base)(nil)
var _ web_types.ResponseStatusProvider = (*Base)(nil)

func NewBaseContext(resp http.ResponseWriter, req *http.Request) *Base {
	if resp == nil || req == nil {
		panic("context: response and request required")
	}
	reqCtx := reqctx.FromContext(req.Context())
	if reqCtx == nil {
		panic("context: request context missing")
	}
	if reqCtx.GetContextValue(BaseContextKey) != nil {
		panic("context: Base already initialized")
	}
	b := &Base{
		RequestContext: reqCtx,
		Resp:           WrapResponseWriter(resp),
		Req:            req.WithContext(reqCtx),
		Data:           reqCtx.GetData(),
	}
	reqCtx.SetContextValue(BaseContextKey, b)
	return b
}

func (b *Base) Written() bool                  { return b.WrittenStatus() != 0 }
func (b *Base) WrittenStatus() int             { return b.Resp.WrittenStatus() }
func (b *Base) Status(status int)              { b.Resp.WriteHeader(status) }
func (b *Base) Write(data []byte) (int, error) { return b.Resp.Write(data) }
func (b *Base) RespHeader() http.Header        { return b.Resp.Header() }
func (b *Base) Header(name, value string)      { b.Resp.Header().Set(name, value) }
func (b *Base) CurrentRequest() *http.Request  { return b.Req }

func (b *Base) JSON(status int, value any) {
	// Encode before committing headers: serialization errors are not partial 2xx responses.
	var body bytes.Buffer
	if err := appjson.MarshalWriteRelaxed(&body, value); err != nil {
		http.Error(b.Resp, "internal server error", http.StatusInternalServerError)
		return
	}
	b.Resp.Header().Set("Content-Type", "application/json;charset=utf-8")
	b.Resp.WriteHeader(status)
	_, _ = b.Resp.Write(body.Bytes())
}
