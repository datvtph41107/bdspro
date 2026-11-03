package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// BenchmarkGinRouting measures only the production framework's parameterized
// route dispatch; it is not evidence about PostgreSQL or full API throughput.
func BenchmarkGinRouting(b *testing.B) {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.GET("/organizations/:id", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})
	request := httptest.NewRequest(http.MethodGet, "/organizations/42", nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		router.ServeHTTP(httptest.NewRecorder(), request)
	}
}
