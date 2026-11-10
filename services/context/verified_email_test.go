package context

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	db "github.com/datvtph41107/bdspro/models/db"
	auth_model "github.com/datvtph41107/bdspro/models/auth"
	"github.com/datvtph41107/bdspro/modules/web"
	"github.com/google/uuid"
	authcrypto "github.com/datvtph41107/bdspro/modules/auth"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type emailVerificationStore struct {
	db.Store
	account db.Account
	err     error
}

func (s emailVerificationStore) GetAccountByID(_ context.Context, id int64) (db.Account, error) {
	if s.err != nil {
		return db.Account{}, s.err
	}
	if id != s.account.ID {
		return db.Account{}, errors.New("missing account")
	}
	return s.account, nil
}

func TestVerifiedEmailGuardsBusinessRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		name       string
		verified   bool
		storeError error
		want       int
	}{
		{"unverified", false, nil, http.StatusForbidden},
		{"verified", true, nil, http.StatusOK},
		{"missing account", false, pgx.ErrNoRows, http.StatusUnauthorized},
		{"store unavailable", false, errors.New("connection unavailable"), http.StatusServiceUnavailable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := web.NewRouter()
			r.AfterRouting(APIContexter())
			r.GinEngine().Use(func(c *gin.Context) {
				c.Request = c.Request.WithContext(db.WithStore(c.Request.Context(), emailVerificationStore{
					account: db.Account{ID: 3, IsEmailVerified: tc.verified},
					err: tc.storeError,
				}))
				c.Next()
			})
			r.Get("/business", func(c *APIContext) {
				sessionID := uuid.New()
				c.Doer = &authcrypto.Payload{AccountID: 3, SessionID: sessionID}
				c.PublishVerifiedActor(auth_model.Actor{AccountID: 3, SessionID: sessionID})
			}, VerifiedEmail(), func(c *APIContext) { c.Status(http.StatusOK) })
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/business", nil))
			if w.Code != tc.want {
				t.Fatalf("status %d, want %d", w.Code, tc.want)
			}
			if tc.want == http.StatusUnauthorized && w.Header().Get("WWW-Authenticate") != "Bearer" {
				t.Fatalf("missing Bearer challenge: %q", w.Header().Get("WWW-Authenticate"))
			}
		})
	}
}
