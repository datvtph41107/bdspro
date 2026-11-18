package apiv1

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Public login must never attempt a password lookup when its
// Redis-backed admission dependency is unavailable.
func TestLoginFailsClosedWithoutAdmission(t *testing.T) {
	handler := Routes()
	req := httptest.NewRequest(http.MethodPost, "/users/login",
		strings.NewReader(`{"email":"user@example.com","password":"secret"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%q; want admission unavailable",
			rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "rate_limit_unavailable") {
		t.Fatalf("unexpected response: %s", rr.Body.String())
	}
}

func TestRefreshFailsClosedWithoutAdmission(t *testing.T) {
	handler := Routes()
	req := httptest.NewRequest(http.MethodPost, "/users/refresh",
		strings.NewReader(`{"session_id":"11111111-1111-4111-8111-111111111111","refresh_token":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%q; want admission unavailable", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "rate_limit_unavailable") {
		t.Fatalf("unexpected response: %s", rr.Body.String())
	}
}
