package common

import (
	"net/http"

	"github.com/datvtph41107/bdspro/modules/log"
	"github.com/google/uuid"
)

// RequestID belongs to the HTTP ingress before route matching.
func RequestID() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			requestID := uuid.NewString()
			w.Header().Set("X-Request-ID", requestID)
			requestLogger := log.With().Str("request_id", requestID).Logger()
			next.ServeHTTP(w, req.WithContext(requestLogger.WithContext(req.Context())))
		})
	}
}
