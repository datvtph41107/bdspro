package context

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	db "github.com/datvtph41107/bdspro/models/db"
	authcrypto "github.com/datvtph41107/bdspro/modules/auth"
	"github.com/datvtph41107/bdspro/modules/web"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type sessionAuthorityReaderFake struct {
	db.Store
	session db.Session
	err     error
	calls   int
}

func (f *sessionAuthorityReaderFake) GetSession(_ context.Context, id uuid.UUID) (db.Session, error) {
	f.calls++
	if f.err != nil {
		return db.Session{}, f.err
	}
	if id != f.session.ID {
		return db.Session{}, pgx.ErrNoRows
	}
	return f.session, nil
}

func TestRequireSessionAfterVerifiedBearer(t *testing.T) {
	gin.SetMode(gin.TestMode)
	maker, err := authcrypto.NewJWTMaker(strings.Repeat("s", 32))
	if err != nil {
		t.Fatal(err)
	}
	sid := uuid.New()
	access, _, err := maker.CreateSessionAccessToken(42, time.Minute, sid)
	if err != nil {
		t.Fatal(err)
	}
	unbound, _, err := maker.CreateToken(42, time.Minute, authcrypto.TokenTypeAccess)
	if err != nil {
		t.Fatal(err)
	}
	refresh, _, err := maker.CreateToken(42, time.Minute, authcrypto.TokenTypeRefresh)
	if err != nil {
		t.Fatal(err)
	}
	active := db.Session{
		ID: sid, AccountID: 42,
		ExpiresAt: time.Now().Add(time.Hour),
	}
	for _, tc := range []struct {
		name          string
		token         string
		session       db.Session
		lookupErr     error
		wantStatus    int
		wantCode      string
		wantLookups   int
		wantChallenge string
	}{
		{name: "live session", token: access, session: active, wantStatus: 204, wantLookups: 1},
		{name: "revoked session", token: access, session: func() db.Session { s := active; s.IsBlocked = true; return s }(), wantStatus: 401, wantCode: "invalid_access_token", wantLookups: 1, wantChallenge: `Bearer error="invalid_token"`},
		{name: "expired session", token: access, session: func() db.Session { s := active; s.ExpiresAt = time.Now().Add(-time.Minute); return s }(), wantStatus: 401, wantCode: "invalid_access_token", wantLookups: 1, wantChallenge: `Bearer error="invalid_token"`},
		{name: "different account", token: access, session: func() db.Session { s := active; s.AccountID = 17; return s }(), wantStatus: 401, wantCode: "invalid_access_token", wantLookups: 1, wantChallenge: `Bearer error="invalid_token"`},
		{name: "deleted session", token: access, lookupErr: pgx.ErrNoRows, wantStatus: 401, wantCode: "invalid_access_token", wantLookups: 1, wantChallenge: `Bearer error="invalid_token"`},
		{name: "database unavailable", token: access, lookupErr: errors.New("db password secret must not leak"), wantStatus: 503, wantCode: "identity_store_unavailable", wantLookups: 1},
		{name: "legacy unbound token", token: unbound, session: active, wantStatus: 401, wantCode: "invalid_access_token", wantLookups: 0, wantChallenge: `Bearer error="invalid_token"`},
		{name: "wrong token purpose", token: refresh, session: active, wantStatus: 401, wantCode: "invalid_access_token", wantLookups: 0, wantChallenge: `Bearer error="invalid_token"`},
		{name: "invalid signed token", token: access + "x", session: active, wantStatus: 401, wantCode: "invalid_access_token", wantLookups: 0, wantChallenge: `Bearer error="invalid_token"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &sessionAuthorityReaderFake{session: tc.session, err: tc.lookupErr}
			router := web.NewRouter()
			router.AfterRouting(APIContexter())
			router.GinEngine().Use(func(c *gin.Context) {
				scope := authcrypto.WithMaker(c.Request.Context(), maker)
				scope = db.WithStore(scope, f)
				c.Request = c.Request.WithContext(scope)
				c.Next()
			})
			router.Get("/private", Authenticate(), RequireSession(), func(c *APIContext) {
				c.Status(http.StatusNoContent)
			})
			req := httptest.NewRequest(http.MethodGet, "/private", nil)
			req.Header.Set("Authorization", "Bearer "+tc.token)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			if w.Code != tc.wantStatus || f.calls != tc.wantLookups {
				t.Fatalf("status=%d lookups=%d want %d/%d: %s", w.Code, f.calls, tc.wantStatus, tc.wantLookups, w.Body.String())
			}
			if got := w.Header().Get("WWW-Authenticate"); got != tc.wantChallenge {
				t.Fatalf("WWW-Authenticate %q want %q", got, tc.wantChallenge)
			}
			if tc.wantCode != "" && !strings.Contains(w.Body.String(), `"code":"`+tc.wantCode+`"`) {
				t.Fatalf("missing code %s: %s", tc.wantCode, w.Body.String())
			}
			if strings.Contains(w.Body.String(), "db password secret") {
				t.Fatal("private database error leaked to caller")
			}
		})
	}
}

func TestRequireSessionWithoutReaderFailsClosed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	maker, err := authcrypto.NewJWTMaker(strings.Repeat("s", 32))
	if err != nil {
		t.Fatal(err)
	}
	token, _, err := maker.CreateSessionAccessToken(42, time.Minute, uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	router := web.NewRouter()
	router.AfterRouting(APIContexter())
	router.GinEngine().Use(func(c *gin.Context) {
		c.Request = c.Request.WithContext(authcrypto.WithMaker(c.Request.Context(), maker))
		c.Next()
	})
	router.Get("/private", Authenticate(), RequireSession(), func(c *APIContext) {
		c.Status(http.StatusNoContent)
	})
	req := httptest.NewRequest(http.MethodGet, "/private", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusServiceUnavailable || !strings.Contains(w.Body.String(), "identity_store_unavailable") {
		t.Fatalf("missing authority reader was not fail-closed: %d %s", w.Code, w.Body.String())
	}
}
