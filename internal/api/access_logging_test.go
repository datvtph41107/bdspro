package api

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/datvtph41107/bdspro/internal/api/response"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

func TestAccessLogIsSafeInfoOutcome(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var output bytes.Buffer
	logger := zerolog.New(&output).With().Str("request_id", "request-123").Logger()

	router := gin.New()
	router.Use(accessLogger())
	router.GET("/organizations/:id", func(c *gin.Context) {
		response.Fail(c, response.InternalServerError)
	})

	request := httptest.NewRequest(http.MethodGet, "/organizations/42?token=private-query-value", nil)
	request = request.WithContext(logger.WithContext(request.Context()))
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d", recorder.Code)
	}

	var event map[string]any
	if err := json.Unmarshal(output.Bytes(), &event); err != nil {
		t.Fatalf("unmarshal access event: %v (log=%q)", err, output.String())
	}
	if event["level"] != "info" || event["message"] != "http.request" {
		t.Fatalf("access must be an info outcome: %+v", event)
	}
	if event["path"] != "/organizations/:id" || event["method"] != http.MethodGet ||
		event["error_code"] != "internal_error" || event["request_id"] != "request-123" {
		t.Fatalf("incomplete request correlation: %+v", event)
	}
	if event["status"] != float64(http.StatusInternalServerError) {
		t.Fatalf("status=%v", event["status"])
	}
	if strings.Contains(output.String(), "private-query-value") || strings.Contains(output.String(), "/organizations/42") {
		t.Fatalf("access log exposed raw request identifiers or query secrets: %s", output.String())
	}
}

func TestUnexpectedErrorProducesOneDiagnosticAndOneOutcome(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var output bytes.Buffer
	logger := zerolog.New(&output).With().Str("request_id", "request-456").Logger()

	router := gin.New()
	router.Use(accessLogger())
	router.GET("/fail", func(c *gin.Context) {
		response.Error(c, errors.New("provider-token-that-must-stay-private"))
	})

	request := httptest.NewRequest(http.MethodGet, "/fail", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request.WithContext(logger.WithContext(request.Context())))

	var levels []string
	var messages []string
	scanner := bufio.NewScanner(bytes.NewReader(output.Bytes()))
	for scanner.Scan() {
		var event struct {
			Level     string `json:"level"`
			Message   string `json:"message"`
			RequestID string `json:"request_id"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			t.Fatal(err)
		}
		if event.RequestID != "request-456" {
			t.Fatalf("missing request correlation: %+v", event)
		}
		levels = append(levels, event.Level)
		messages = append(messages, event.Message)
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	if len(levels) != 2 || levels[0] != "error" || messages[0] != "http.application_error" ||
		levels[1] != "info" || messages[1] != "http.request" {
		t.Fatalf("unexpected diagnostic/outcome logs: levels=%v messages=%v", levels, messages)
	}
	if strings.Contains(output.String(), "provider-token-that-must-stay-private") {
		t.Fatalf("private error was logged: %s", output.String())
	}
}
