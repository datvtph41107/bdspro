package web

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-chi/chi/v5"
)

// BenchmarkRouterCompare uses the same method, parameterized route, request
// and ResponseRecorder for each framework. It measures only HTTP routing and
// handler dispatch; authentication, SQLC and logging must be tested separately.
func BenchmarkRouterCompare(b *testing.B) {
	req := httptest.NewRequest(http.MethodGet, "/organizations/42", nil)

	b.Run("Gin", func(b *testing.B) {
		gin.SetMode(gin.ReleaseMode)
		r := gin.New()
		r.GET("/organizations/:id", func(c *gin.Context) {
			c.Status(http.StatusNoContent)
		})
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			r.ServeHTTP(httptest.NewRecorder(), req)
		}
	})

	b.Run("ChiNative", func(b *testing.B) {
		r := chi.NewRouter()
		r.Get("/organizations/{id}", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		})
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			r.ServeHTTP(httptest.NewRecorder(), req)
		}
	})

	b.Run("GiteaStyle", func(b *testing.B) {
		r := NewRouter()
		r.Get("/organizations/{id}", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		})
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			r.ServeHTTP(httptest.NewRecorder(), req)
		}
	})
}
