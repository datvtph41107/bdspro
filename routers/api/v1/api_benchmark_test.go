package apiv1

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Baseline is the real C01/C02 route tree, with only its health route.
// Its middleware and response path differ from the isolated spikes.
func BenchmarkC01Health(b *testing.B) {
	router := Routes()
	router.Compile()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			b.Fatalf("status=%d", w.Code)
		}
	}
}
