package routers

import (
	"net/http"
	"time"

	"github.com/datvtph41107/bdspro/modules/log"
	"github.com/datvtph41107/bdspro/modules/reqctx"
	appctx "github.com/datvtph41107/bdspro/services/context"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
)

// requestLogging records response metadata, never bodies, cookies or query strings.
func requestLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		id := uuid.NewString()
		logger := log.With().Str("request_id", id).Logger()
		ctx, finish := reqctx.NewRequestContext(logger.WithContext(r.Context()))
		defer finish()
		w.Header().Set("X-Request-ID", id)
		response := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(response, r.WithContext(ctx))

		status := response.Status()
		if status == 0 {
			status = http.StatusOK
		}
		event := logger.Info()
		if status >= 500 {
			event = logger.Error()
		} else if status >= 400 {
			event = logger.Warn()
		}
		data := reqctx.FromContext(ctx).GetData()
		for _, field := range []struct{ key, name string }{
			{appctx.ErrorCodeKey, "code"},
			{appctx.ErrorMessageKey, "error"},
			{appctx.ErrorReasonKey, "reason"},
		} {
			if value, ok := data[field.key].(string); ok && value != "" {
				event.Str(field.name, value)
			}
		}
		event.Str("component", "http").Str("method", r.Method).Str("path", r.URL.Path).Int("status", status).Int("bytes", response.BytesWritten()).
			Dur("duration_ms", time.Since(started)).Msg("http.request")
	})
}
