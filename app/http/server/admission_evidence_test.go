package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/datvtph41107/bdspro/app/http/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// ARC-03 uses the real Redis-backed middleware but a synthetic route: it must
// never call the real login handler, create sessions or write PostgreSQL data.
func TestARC03SourceAdmissionPrecedesJSONParsing(t *testing.T) {
	client := redisForAuthRateTest(t)
	server := newTestServer(t, nil)
	require.NoError(t, server.EnableAuthRateLimiting(client))
	operation := "arc03-malformed-" + uuid.NewString()
	reachedHandler := false
	server.router.POST("/test/arc03/source", server.IPActiveSource(operation, 2, time.Minute),
		middleware.PreLogin(), func(ctx *gin.Context) {
			reachedHandler = true
			ctx.Status(http.StatusNoContent)
		})

	send := func() *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/test/arc03/source", strings.NewReader("{"))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "192.0.2.61:43210"
		res := httptest.NewRecorder()
		server.Handler().ServeHTTP(res, req)
		return res
	}

	require.Equal(t, http.StatusBadRequest, send().Code)
	require.Equal(t, http.StatusBadRequest, send().Code)
	denied := send()
	require.Equal(t, http.StatusTooManyRequests, denied.Code)
	require.Equal(t, "60", denied.Header().Get("Retry-After"))
	require.False(t, reachedHandler)
}

func TestARC03IdentityAdmissionAcrossClientIPs(t *testing.T) {
	client := redisForAuthRateTest(t)
	server := newTestServer(t, nil)
	require.NoError(t, server.EnableAuthRateLimiting(client))
	operation := "arc03-identity-" + uuid.NewString()
	server.router.POST("/test/arc03/identity",
		server.IPActiveSource(operation, 30, time.Minute),
		middleware.PreLogin(),
		server.IPActive(operation, 30, 2, time.Minute, middleware.LoginIdentity),
		func(ctx *gin.Context) { ctx.Status(http.StatusNoContent) },
	)
	send := func(remote, email string) int {
		body := `{"email":"` + email + `","password":"some-password"}`
		req := httptest.NewRequest(http.MethodPost, "/test/arc03/identity", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = remote
		res := httptest.NewRecorder()
		server.Handler().ServeHTTP(res, req)
		return res.Code
	}

	require.Equal(t, http.StatusNoContent, send("192.0.2.70:3456", "Alice@Example.test"))
	require.Equal(t, http.StatusNoContent, send("192.0.2.71:3456", "alice@example.test"))
	require.Equal(t, http.StatusTooManyRequests, send("192.0.2.72:3456", "ALICE@EXAMPLE.TEST"))
}

func TestARC03SourceQuotaAtomicAcrossConcurrentRequests(t *testing.T) {
	client := redisForAuthRateTest(t)
	server := newTestServer(t, nil)
	require.NoError(t, server.EnableAuthRateLimiting(client))
	operation := "arc03-concurrent-" + uuid.NewString()
	const allowed = 4
	const attempts = 20
	server.router.GET("/test/arc03/concurrent", server.IPActiveSource(operation, allowed, time.Minute),
		func(ctx *gin.Context) { ctx.Status(http.StatusNoContent) })

	var wg sync.WaitGroup
	results := make(chan int, attempts)
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, "/test/arc03/concurrent", nil)
			req.RemoteAddr = "192.0.2.80:4567"
			res := httptest.NewRecorder()
			server.Handler().ServeHTTP(res, req)
			results <- res.Code
		}()
	}
	wg.Wait()
	close(results)
	counts := map[int]int{}
	for code := range results {
		counts[code]++
	}
	require.Equal(t, allowed, counts[http.StatusNoContent], "only the first quota of concurrent requests may pass")
	require.Equal(t, attempts-allowed, counts[http.StatusTooManyRequests])
	require.Len(t, counts, 2, "unexpected response, e.g. Redis unavailable")
}
