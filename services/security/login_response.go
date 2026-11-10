package security

import (
	"net/http"

	"github.com/datvtph41107/bdspro/modules/log"
)

// loginResponseStatus observes the actual HTTP status emitted by the typed
// action or preceding admission middleware. It preserves underlying transport
// capabilities through Unwrap rather than creating another response writer.
type loginResponseStatus struct {
	http.ResponseWriter
	status int
}

func (w *loginResponseStatus) Unwrap() http.ResponseWriter { return w.ResponseWriter }

func (w *loginResponseStatus) WriteHeader(status int) {
	if w.status != 0 {
		return
	}
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *loginResponseStatus) Write(p []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(p)
}

// LogLoginResponse is an HTTP-native middleware provider. It records the
// result even when IPActive or credential throttling rejects before Login.
func LogLoginResponse() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			recorder := &loginResponseStatus{ResponseWriter: w}
			next.ServeHTTP(recorder, req)
			status := recorder.status
			if status == 0 {
				status = http.StatusOK
			}
			event := log.Ctx(req.Context()).Info()
			outcome := "succeeded"
			switch {
			case status == http.StatusUnauthorized:
				outcome = "rejected"
				event = log.Ctx(req.Context()).Warn()
			case status == http.StatusTooManyRequests:
				outcome = "throttled"
				event = log.Ctx(req.Context()).Warn()
			case status >= 500:
				outcome = "failed"
			case status < 200 || status >= 300:
				outcome = "rejected"
				event = log.Ctx(req.Context()).Warn()
			}
			event.Str("outcome", outcome).Int("status", status).Msg("auth.login_completed")
		})
	}
}
