package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/datvtph41107/bdspro/internal/api/response"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

func TestRequestMonitorDoesNotWarnOnSingleMalformedRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var output bytes.Buffer
	logger := zerolog.New(&output).Level(zerolog.InfoLevel)

	router := gin.New()
	router.Use(RequestMonitor(nil))
	router.GET("/test", func(c *gin.Context) {
		response.Fail(c, response.InvalidRequest)
	})
	request := httptest.NewRequest(http.MethodGet, "/test", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request.WithContext(logger.WithContext(request.Context())))

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", recorder.Code)
	}
	if strings.Contains(output.String(), "security.request_denied") {
		t.Fatalf("a single invalid request was logged as security warning: %s", output.String())
	}
}

func TestRequestMonitorWarnsOnDeniedAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var output bytes.Buffer
	logger := zerolog.New(&output).Level(zerolog.InfoLevel)

	router := gin.New()
	router.Use(RequestMonitor(nil))
	router.GET("/me", func(c *gin.Context) {
		response.Fail(c, response.BearerRequired)
	})
	request := httptest.NewRequest(http.MethodGet, "/me", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request.WithContext(logger.WithContext(request.Context())))

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d", recorder.Code)
	}
	if !strings.Contains(output.String(), `"message":"security.request_denied"`) ||
		!strings.Contains(output.String(), `"reason":"authentication"`) {
		t.Fatalf("missing authentication denial event: %s", output.String())
	}
}
