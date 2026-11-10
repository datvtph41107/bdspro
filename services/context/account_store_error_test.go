package context

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	db "github.com/datvtph41107/bdspro/models/db"
	auth_model "github.com/datvtph41107/bdspro/models/auth"
	"github.com/datvtph41107/bdspro/modules/web"
	"github.com/google/uuid"
	authcrypto "github.com/datvtph41107/bdspro/modules/auth"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

func TestVerifiedContactWithoutStoreFailsClosed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := web.NewRouter()
	router.AfterRouting(APIContexter())
	router.Get("/orgs", func(c *APIContext) {
		id := uuid.New()
		c.Doer = &authcrypto.Payload{AccountID: 7, SessionID: id}
		c.PublishVerifiedActor(auth_model.Actor{AccountID: 7, SessionID: id})
	}, VerifiedContact(), func(c *APIContext) {
		c.Status(http.StatusOK)
	})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/orgs", nil))
	if w.Code != http.StatusServiceUnavailable || !strings.Contains(w.Body.String(), `"code":"identity_store_unavailable"`) {
		t.Fatalf("unexpected failure status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestAccountStoreErrorDoesNotExposePrivateDetails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var output bytes.Buffer
	logger := zerolog.New(&output)
	const private = "sensitive-database-credential"
	router := web.NewRouter()
	router.AfterRouting(APIContexter())
	router.GinEngine().Use(func(c *gin.Context) {
		scope := logger.WithContext(c.Request.Context())
		scope = db.WithStore(scope, emailVerificationStore{
			account: db.Account{ID: 7},
			err: errors.New(private),
		})
		c.Request = c.Request.WithContext(scope)
		c.Next()
	})
	router.Get("/business", func(c *APIContext) {
		id := uuid.New()
		c.Doer = &authcrypto.Payload{AccountID: 7, SessionID: id}
		c.PublishVerifiedActor(auth_model.Actor{AccountID: 7, SessionID: id})
	}, VerifiedEmail(), func(c *APIContext) { c.Status(http.StatusOK) })
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/business", nil))
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d want=%d", w.Code, http.StatusServiceUnavailable)
	}
	if strings.Contains(w.Body.String(), private) || strings.Contains(output.String(), private) {
		t.Fatalf("private data was exposed: body=%s logs=%s", w.Body.String(), output.String())
	}
	if !strings.Contains(output.String(), `"dependency":"postgres"`) {
		t.Fatalf("missing dependency diagnostic: %s", output.String())
	}
}
