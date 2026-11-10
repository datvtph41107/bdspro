package context

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	organization_model "github.com/datvtph41107/bdspro/models/organization"
	"github.com/datvtph41107/bdspro/modules/util"
	"github.com/gin-gonic/gin"
	"github.com/datvtph41107/bdspro/modules/web"
	"github.com/jackc/pgx/v5/pgconn"
)

// API error identities must survive model wrapping without exposing driver details.
func TestModelCauseUsesStablePublicResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cause := &pgconn.PgError{
		Code: "23505", ConstraintName: "organization_members_one_open_uq",
		Message: "password=private-db-secret",
	}
	err := util.WithCause(organization_model.ErrMembershipExists, cause)
	if err.Error() != organization_model.ErrMembershipExists.Error() {
		t.Fatal("model leaked an internal PostgreSQL message")
	}
	w := httptest.NewRecorder()
	router := web.NewRouter()
	router.AfterRouting(APIContexter())
	router.Post("/", func(ctx *APIContext) {
		ctx.APIErrorAuto(err)
	})
	router.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/", nil))
	if w.Code != http.StatusConflict || !strings.Contains(w.Body.String(), "organization.membership_exists") {
		t.Fatalf("unexpected HTTP contract: %d %s", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), "private-db-secret") || strings.Contains(w.Body.String(), "23505") {
		t.Fatalf("database detail leaked in response: %s", w.Body.String())
	}
}
