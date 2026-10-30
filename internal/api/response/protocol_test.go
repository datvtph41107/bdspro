package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSharedProtocolResponses(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name, code, message, challenge, retryAfter string
		status                                     int
		write                                      func(*gin.Context)
	}{
		{"missing bearer", "unauthenticated", "missing or invalid bearer token", "Bearer", "", 401, BearerRequired},
		{"invalid bearer", "invalid_access_token", "invalid access token", `Bearer error="invalid_token"`, "", 401, InvalidAccessToken},
		{"missing principal", "unauthenticated", "authentication required", "Bearer", "", 401, AuthenticationRequired},
		{"rate limited", "rate_limited", "too many requests", "", "60", 429, func(c *gin.Context) { RateLimited(c, 60) }},
		{"limiter unavailable", "rate_limit_unavailable", "authentication temporarily unavailable", "", "", 503, RateLimitUnavailable},
		{"internal error", "internal_error", "internal server error", "", "", 500, InternalError},
		{"endpoint local failure", "google_invalid_state", "invalid Google authorization state", "", "", 400, func(c *gin.Context) {
			Reject(c, http.StatusBadRequest, "google_invalid_state", "invalid Google authorization state")
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
			tt.write(c)
			if w.Code != tt.status {
				t.Fatalf("status %d, want %d", w.Code, tt.status)
			}
			var got struct {
				Code  string `json:"code"`
				Error string `json:"error"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
				t.Fatal(err)
			}
			if got.Code != tt.code || got.Error != tt.message {
				t.Fatalf("unexpected public contract: %+v", got)
			}
			if got := w.Header().Get("WWW-Authenticate"); got != tt.challenge {
				t.Fatalf("WWW-Authenticate %q, want %q", got, tt.challenge)
			}
			if got := w.Header().Get("Retry-After"); got != tt.retryAfter {
				t.Fatalf("Retry-After %q, want %q", got, tt.retryAfter)
			}
			if c.GetString(ErrorCodeKey) != tt.code {
				t.Fatalf("request error code %q, want %q", c.GetString(ErrorCodeKey), tt.code)
			}
		})
	}
}
