package context

import (
	"errors"
	"github.com/datvtph41107/bdspro/modules/reqctx"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apperrors "github.com/datvtph41107/bdspro/modules/errors"
)

func TestApplicationErrorCompatibility(t *testing.T) {
	cases := []struct {
		err    *apperrors.Error
		status int
		code   string
	}{
		{apperrors.NameRequired, 400, "http.name_required"},
		{apperrors.OrganizationNotFound, 404, "organization.not_found"},
		{apperrors.OrganizationOwnerRequired, 403, "organization.owner_required"},
		{apperrors.MembershipExists, 409, "organization.membership_exists"},
		{apperrors.OTPInvalid, 401, "auth.otp_invalid"},
		{apperrors.OTPCooldown, 429, "auth.otp_cooldown"},
	}
	for _, tc := range cases {
		status, code, _, ok := applicationError(apperrors.Wrap(tc.err, errors.New("secret=password")))
		if !ok || status != tc.status || code != tc.code {
			t.Errorf("mapping %v => %d %s %v", tc.err, status, code, ok)
		}
	}
}

func TestRateLimitAndBearerContract(t *testing.T) {
	tests := []struct {
		name   string
		write  func(*APIContext)
		header string
		value  string
		status int
		code   string
	}{
		{"bearer", (*APIContext).BearerRequired, "WWW-Authenticate", "Bearer", http.StatusUnauthorized, "unauthenticated"},
		{"rate", func(c *APIContext) { c.RateLimited(31) }, "Retry-After", "31", http.StatusTooManyRequests, "rate_limited"},
		{"unavailable", (*APIContext).IdentityStoreUnavailable, "", "", http.StatusServiceUnavailable, "identity_store_unavailable"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			reqctx.RequestContextHandler()(APIContexter()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { tc.write(GetAPIContext(r)) }))).ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
			if w.Code != tc.status || !strings.Contains(w.Body.String(), tc.code) || (tc.header != "" && w.Header().Get(tc.header) != tc.value) {
				t.Fatalf("got %d %v %s", w.Code, w.Header(), w.Body.String())
			}
		})
	}
}
