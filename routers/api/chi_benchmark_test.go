package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

// BenchmarkChiRouting measures route dispatch alone, not full API throughput.
func BenchmarkChiRouting(b *testing.B) {
	router := chi.NewRouter()
	router.Get("/orgs/{id}", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	req := httptest.NewRequest(http.MethodGet, "/orgs/42", nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		router.ServeHTTP(httptest.NewRecorder(), req)
	}
}
