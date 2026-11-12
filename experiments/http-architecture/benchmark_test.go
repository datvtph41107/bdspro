package spike

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Microbenchmarks measure only this shared probe; they do not predict
// production p95/p99, allocation profiles under load, or migration cost.
func BenchmarkHealth(b *testing.B) {
	for _, router := range routers {
		b.Run(router.name, func(b *testing.B) {
			h := router.new()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				w := httptest.NewRecorder()
				h.ServeHTTP(w, req)
				if w.Code != http.StatusOK {
					b.Fatalf("status=%d", w.Code)
				}
			}
		})
	}
}
