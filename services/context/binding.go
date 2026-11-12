package context

import (
	"errors"
	"net/http"

	"github.com/datvtph41107/bdspro/modules/web"
	"github.com/datvtph41107/bdspro/modules/web/middleware"
)

// maxJSONBodyBytes is the API transport limit, applied before JSON decoding.
// A configurable limit can replace it when a real endpoint requires one.
const maxJSONBodyBytes int64 = 1 << 20

// BindJSON decodes a single JSON document and stops the typed handler chain
// on an error. The API context owns error formatting, not the web primitive.
func BindJSON[T any]() func(*APIContext) {
	return func(ctx *APIContext) {
		ctx.Req.Body = http.MaxBytesReader(ctx.Resp, ctx.Req.Body, maxJSONBodyBytes)
		defer ctx.Req.Body.Close()

		form, err := middleware.BindJSON[T](middleware.RequestJSONBinder{Request: ctx.Req})
		if err != nil {
			var tooLarge *http.MaxBytesError
			if errors.As(err, &tooLarge) {
				ctx.APIError(http.StatusRequestEntityTooLarge, "request body too large")
				return
			}
			ctx.APIError(http.StatusBadRequest, "invalid request body")
			return
		}
		web.SetForm(ctx, form)
	}
}
