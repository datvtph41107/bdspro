package response

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	apperrors "github.com/datvtph41107/bdspro/internal/errors"
	"github.com/gin-gonic/gin"
)

func TestPublicErrorContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cases := []struct {
		name          string
		err           error
		invalid       bool
		status        int
		code, message string
	}{
		{"invalid", errors.New("private validation detail"), true, 400, "invalid_request", "invalid request"},
		{"too_large", &http.MaxBytesError{Limit: 1024}, true, 413, "request_body_too_large", "request body too large"},
		{"domain", apperrors.NameRequired, false, 400, "http.name_required", "name required"},
		{"wrapped", apperrors.Wrap(apperrors.MembershipExists, errors.New("private DB detail")), false, 409, "organization.membership_exists", "membership already exists"},
		{"otp", apperrors.OTPInvalid, false, 401, "auth.otp_invalid", "invalid or expired code"},
		{"unknown", errors.New("postgres password secret"), false, 500, "internal_error", "internal server error"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
			if tc.invalid {
				Invalid(c, tc.err)
			} else {
				Error(c, tc.err)
			}
			if w.Code != tc.status {
				t.Fatalf("status=%d want=%d", w.Code, tc.status)
			}
			var got struct {
				Error string `json:"error"`
				Code  string `json:"code"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
				t.Fatal(err)
			}
			if got.Error != tc.message || got.Code != tc.code {
				t.Fatalf("response=%+v want error=%q code=%q", got, tc.message, tc.code)
			}
			if c.GetString(ErrorCodeKey) != tc.code {
				t.Fatal("missing monitor error code")
			}
		})
	}
}
