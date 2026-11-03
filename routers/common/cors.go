package common

import (
	"net/http"

	"github.com/rs/cors"
)

// CORS is a protocol boundary shared by HTTP transports; policy comes from settings.
func CORS(handler http.Handler, allowedOrigins []string) http.Handler {
	return cors.New(cors.Options{
		AllowedOrigins: allowedOrigins,
		AllowedMethods: []string{http.MethodHead, http.MethodOptions, http.MethodGet, http.MethodPost,
			http.MethodPut, http.MethodPatch, http.MethodDelete},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		ExposedHeaders:   []string{"X-Request-ID"},
		AllowCredentials: true,
	}).Handler(handler)
}
