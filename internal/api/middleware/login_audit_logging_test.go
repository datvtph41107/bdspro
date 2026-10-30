package middleware

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/datvtph41107/bdspro/internal/api/response"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

func TestLoginAuditDoesNotDuplicateServerError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var output bytes.Buffer
	logger := zerolog.New(&output).With().Str("request_id", "req-login-1").Logger()

	router := gin.New()
	router.POST("/users/login", LoginAudit(), func(c *gin.Context) {
		response.Fail(c, response.RateLimitUnavailable)
	})

	request := httptest.NewRequest(http.MethodPost, "/users/login?password=private-query-secret", strings.NewReader("private-body"))
	request = request.WithContext(logger.WithContext(request.Context()))
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d", recorder.Code)
	}

	var event struct {
		Level     string `json:"level"`
		Outcome   string `json:"outcome"`
		Message   string `json:"message"`
		RequestID string `json:"request_id"`
	}
	if err := json.Unmarshal(output.Bytes(), &event); err != nil {
		t.Fatal(err)
	}
	if event.Level != "info" || event.Outcome != "failed" ||
		event.Message != "auth.login_completed" || event.RequestID != "req-login-1" {
		t.Fatalf("unexpected audit event: %+v", event)
	}
	if strings.Contains(output.String(), "private-query-secret") ||
		strings.Contains(output.String(), "private-body") {
		t.Fatalf("authentication audit exposed private request data: %s", output.String())
	}
}
