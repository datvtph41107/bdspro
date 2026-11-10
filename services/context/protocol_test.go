package context

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/datvtph41107/bdspro/modules/web"
	"github.com/gin-gonic/gin"
)

func TestSharedProtocolResponses(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name, code, message, challenge, retryAfter string
		status                                     int
		write                                      func(*APIContext)
	}{
		{"missing bearer", "unauthenticated", "missing or invalid bearer token", "Bearer", "", 401, func(c *APIContext) {
			c.Header("WWW-Authenticate", "Bearer")
			c.APIError(http.StatusUnauthorized, "unauthenticated", "missing or invalid bearer token")
		}},
		{"invalid bearer", "invalid_access_token", "invalid access token", `Bearer error="invalid_token"`, "", 401, func(c *APIContext) {
			c.Header("WWW-Authenticate", `Bearer error="invalid_token"`)
			c.APIError(http.StatusUnauthorized, "invalid_access_token", "invalid access token")
		}},
		{"missing principal", "unauthenticated", "authentication required", "Bearer", "", 401, func(c *APIContext) { c.AuthenticationRequired() }},
		{"rate limited", "rate_limited", "too many requests", "", "60", 429, func(c *APIContext) {
			c.Header("Retry-After", "60")
			c.APIError(http.StatusTooManyRequests, "rate_limited", "too many requests")
		}},
		{"limiter unavailable", "rate_limit_unavailable", "authentication temporarily unavailable", "", "", 503, func(c *APIContext) {
			c.APIError(http.StatusServiceUnavailable, "rate_limit_unavailable", "authentication temporarily unavailable")
		}},
		{"internal error", "internal_error", "internal server error", "", "", 500, func(c *APIContext) {
			c.APIError(http.StatusInternalServerError, "internal_error", "internal server error")
		}},
		{"endpoint local failure", "google_invalid_state", "invalid Google authorization state", "", "", 400, func(c *APIContext) {
			c.APIError(http.StatusBadRequest, "google_invalid_state", "invalid Google authorization state")
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			router := web.NewRouter()
			router.AfterRouting(APIContexter())
			router.Get("/", tt.write)
			var trace *gin.Context
			router.GinEngine().Use(func(c *gin.Context) { trace = c; c.Next() })
			router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
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
			if trace == nil || trace.GetString(ErrorCodeKey) != tt.code {
				t.Fatalf("request error code %q, want %q", func() string {
					if trace == nil {
						return ""
					}
					return trace.GetString(ErrorCodeKey)
				}(), tt.code)
			}
		})
	}
}
