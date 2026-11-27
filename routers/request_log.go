package routers

import (
	"net/http"
	"sync/atomic"
	"time"

	"github.com/datvtph41107/bdspro/modules/log"
	"github.com/datvtph41107/bdspro/modules/reqctx"
	"github.com/datvtph41107/bdspro/modules/web"
	appctx "github.com/datvtph41107/bdspro/services/context"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// requestLogging records response metadata, never bodies, cookies or query strings.
func requestLogging(next http.Handler) http.Handler {
	return requestLoggingWithThreshold(next, 3*time.Second)
}

func requestLoggingWithThreshold(next http.Handler, threshold time.Duration) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		id := uuid.NewString()
		logger := log.With().Str("request_id", id).Logger()
		ctx, finish := reqctx.NewRequestContext(logger.WithContext(r.Context()))
		defer finish()
		w.Header().Set("X-Request-ID", id)
		ctx, activity := web.WithRequestActivity(ctx)
		response := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		method, path := r.Method, r.URL.Path
		entry := func(event *zerolog.Event) *zerolog.Event {
			event = event.Str("component", "http").Str("method", method).Str("path", path)
			if info := activity.LastHandler(); info != nil {
				event.Str("handler", info.Name)
				if info.Source != "" {
					event.Str("handler_source", info.Source)
				}
			}
			return event
		}
		entry(logger.Debug()).Msg("http.started")

		// The timer never reads the response writer, request or request data map.
		// Wait for a running callback before writing the terminal event.
		var finished atomic.Bool
		slowDone := make(chan struct{})
		timer := time.AfterFunc(threshold, func() {
			defer close(slowDone)
			if finished.Load() {
				return
			}
			entry(logger.Warn()).Dur("duration_ms", time.Since(started)).Msg("http.slow")
		})
		defer func() {
			finished.Store(true)
			elapsed := time.Since(started)
			if !timer.Stop() {
				<-slowDone
			}
			if recovered := recover(); recovered != nil {
				// Only the HTTP boundary translates a panic. net/http logs arbitrary
				// panic values unless the exact ErrAbortHandler sentinel is used.
				// Preserve abort semantics without reporting a false HTTP status.
				level, reason := log.ERROR, "panic"
				if recovered == http.ErrAbortHandler {
					level, reason = log.WARN, "handler_abort"
				}
				event := entry(logger.WithLevel(level)).Dur("duration_ms", elapsed).
					Str("reason", reason)
				if status := response.Status(); status != 0 {
					event.Int("status", status)
				}
				if reason == "panic" {
					log.StackDetails(event)
				}
				event.Msg("http.failed")
				panic(http.ErrAbortHandler)
			}

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
			entry(event).Int("status", status).Int("bytes", response.BytesWritten()).
				Dur("duration_ms", elapsed).Msg("http.request")
		}()
		next.ServeHTTP(response, r.WithContext(ctx))
	})
}
