package response

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apperrors "github.com/datvtph41107/bdspro/internal/core/errors"
	"github.com/gin-gonic/gin"
)

func TestInvalidResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		name string
		err error
		status int
		message string
	}{
		{"malformed", errors.New("private validator detail"), http.StatusBadRequest, "invalid request"},
		{"too large", &http.MaxBytesError{Limit: 1024}, http.StatusRequestEntityTooLarge, "request body too large"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, w := gin.CreateTestContext(httptest.NewRecorder())
			Invalid(c, tc.err)
			if w.Code != tc.status || strings.TrimSpace(w.Body.String()) != `{"error":"`+tc.message+`"}` {
				t.Fatalf("status=%d response=%s", w.Code, w.Body.String())
			}
		})
	}
}

func TestApplicationErrorResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		name string
		err error
		status int
		message string
	}{
		{"business", apperrors.NameRequired, http.StatusBadRequest, "name required"},
		{"wrapped", apperrors.Wrap(apperrors.MembershipExists, errors.New("private unique constraint")), http.StatusConflict, "membership already exists"},
		{"unknown", errors.New("database password is private"), http.StatusInternalServerError, "internal server error"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, w := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
			Error(c, tc.err)
			if w.Code != tc.status || strings.TrimSpace(w.Body.String()) != `{"error":"`+tc.message+`"}` {
				t.Fatalf("status=%d response=%s", w.Code, w.Body.String())
			}
		})
	}
}
