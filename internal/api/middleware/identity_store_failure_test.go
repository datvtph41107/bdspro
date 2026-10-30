package middleware

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	db "github.com/datvtph41107/bdspro/internal/database"
	"github.com/datvtph41107/bdspro/internal/token"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

func TestVerifiedContactWithoutStoreFailsClosed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/organizations", VerifiedContact(nil), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/organizations", nil))
	if w.Code != http.StatusServiceUnavailable || !strings.Contains(w.Body.String(), `"code":"identity_store_unavailable"`) {
		t.Fatalf("unexpected failure status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestIdentityStoreErrorDoesNotExposePrivateDetails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var output bytes.Buffer
	logger := zerolog.New(&output)
	const private = "sensitive-database-credential"
	router := gin.New()
	router.GET("/business",
		func(c *gin.Context) {
			c.Set(AuthorizationPayloadKey, &token.Payload{AccountID: 7})
			c.Request = c.Request.WithContext(logger.WithContext(c.Request.Context()))
			c.Next()
		},
		VerifiedEmail(emailVerificationStore{
			account: db.Account{ID: 7},
			err:     errors.New(private),
		}),
		func(c *gin.Context) { c.Status(http.StatusOK) },
	)
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
