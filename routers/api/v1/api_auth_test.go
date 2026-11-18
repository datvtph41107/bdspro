package apiv1

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLogoutRequiresVerifiedActor(t *testing.T) {
	for _, authorization := range []string{"", "Bearer"} {
		t.Run(authorization, func(t *testing.T) {
			handler := Routes()
			req := httptest.NewRequest(http.MethodPost, "/users/me/logout", nil)
			if authorization != "" {
				req.Header.Set("Authorization", authorization)
			}
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
			if rr.Code != http.StatusUnauthorized {
				t.Fatalf("unauthenticated logout: status=%d body=%q", rr.Code, rr.Body.String())
			}
		})
	}
}
