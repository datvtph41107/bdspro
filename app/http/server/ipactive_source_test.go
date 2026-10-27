package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/datvtph41107/bdspro/app/http/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestIPActiveSourceRunsBeforePreLogin(t *testing.T) {
	client := redisForAuthRateTest(t)
	server := newTestServer(t, nil)
	require.NoError(t, server.EnableAuthRateLimiting(client))
	action := "early-" + uuid.NewString()
	server.router.POST("/test/prelogin", server.IPActiveSource(action, 2, time.Minute),
		middleware.PreLogin(), func(ctx *gin.Context) { ctx.Status(http.StatusNoContent) })
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodPost, "/test/prelogin",
			strings.NewReader(`{"email":"not-valid","password":"test"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		if i < 2 {
			require.Equal(t, http.StatusBadRequest, w.Code)
		} else {
			require.Equal(t, http.StatusTooManyRequests, w.Code)
		}
	}
}
