package user

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/datvtph41107/bdspro/modules/web"
	appctx "github.com/datvtph41107/bdspro/services/context"
)

func TestVerificationGETOnlyShowsConfirmation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		path string
		want int
	}{
		{"/users/verify_email?email_id=7&secret_code=test-token", http.StatusOK},
		{"/users/verify_email?email_id=0", http.StatusBadRequest},
	} {
		t.Run(tc.path, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			router := web.NewRouter()
			router.AfterRouting(appctx.APIContexter())
			router.Get("/users/verify_email", VerifyEmailPage)
			// No AuthService or store exists: GET must not attempt consumption.
			router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, tc.path, nil))
			if recorder.Code != tc.want {
				t.Fatalf("status=%d want=%d", recorder.Code, tc.want)
			}
			if tc.want == http.StatusOK {
				if !strings.Contains(recorder.Body.String(), `method="post"`) ||
					!strings.Contains(recorder.Body.String(), `name="secret_code"`) {
					t.Fatalf("GET must render an explicit POST confirmation form")
				}
			}
		})
	}
}
