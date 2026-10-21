package gapi

import (
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (w *statusRecorder) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func HttpLogger(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		writer := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		handler.ServeHTTP(writer, r)
		log.Info().Str("method", r.Method).Str("path", r.URL.Path).
			Int("status", writer.status).Dur("duration", time.Since(start)).Msg("HTTP request")
	})
}
