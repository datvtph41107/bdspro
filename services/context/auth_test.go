package context

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	authcrypto "github.com/datvtph41107/bdspro/modules/auth"
	"github.com/gin-gonic/gin"
)

func TestAuthenticateAndRequestLocalDoer(t *testing.T) {
	gin.SetMode(gin.TestMode)
	maker, err := authcrypto.NewJWTMaker(strings.Repeat("x", 32))
	if err != nil {
		t.Fatal(err)
	}
	access, _, err := maker.CreateToken(42, time.Minute, authcrypto.TokenTypeAccess)
	if err != nil {
		t.Fatal(err)
	}
	refresh, _, err := maker.CreateToken(42, time.Minute, authcrypto.TokenTypeRefresh)
	if err != nil {
		t.Fatal(err)
	}

	r := gin.New()
	reached := 0
	r.GET("/me", Authenticate(maker), Wrap(func(c *APIContext) {
		reached++
		if c.Doer == nil || c.Doer.AccountID != 42 {
			t.Fatalf("missing verified principal: %+v", c.Doer)
		}
		if p, ok := c.SignedPrincipal(); !ok || p != c.Doer {
			t.Fatal("context principal did not come from Authenticate")
		}
		c.JSON(http.StatusOK, gin.H{"account_id": c.Doer.AccountID})
	}))

	cases := []struct {
		name, authorization, code, challenge string
		status                               int
	}{
		{name: "missing", status: 401, code: "unauthenticated", challenge: "Bearer"},
		{name: "wrong scheme", authorization: "Basic abc", status: 401, code: "unauthenticated", challenge: "Bearer"},
		{name: "refresh", authorization: "Bearer " + refresh, status: 401, code: "invalid_access_token", challenge: `Bearer error="invalid_token"`},
		{name: "valid", authorization: "Bearer " + access, status: 200},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			before := reached
			req := httptest.NewRequest(http.MethodGet, "/me", nil)
			if tc.authorization != "" {
				req.Header.Set("Authorization", tc.authorization)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != tc.status || w.Header().Get("WWW-Authenticate") != tc.challenge {
				t.Fatalf("status=%d challenge=%q want=%d %q", w.Code, w.Header().Get("WWW-Authenticate"), tc.status, tc.challenge)
			}
			if tc.status != http.StatusOK {
				if reached != before {
					t.Fatal("invalid principal reached endpoint")
				}
				var body struct {
					Code string `json:"code"`
				}
				if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil || body.Code != tc.code {
					t.Fatalf("code=%q parse=%v want=%q", body.Code, err, tc.code)
				}
			} else if reached != before+1 {
				t.Fatal("valid principal did not reach endpoint")
			}
		})
	}
}

func TestWrapWithoutAuthenticationHasNoDoer(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/public", Wrap(func(c *APIContext) {
		if c.Doer != nil {
			t.Fatal("unauthenticated request has a forged principal")
		}
		c.Status(http.StatusNoContent)
	}))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/public", nil))
	if w.Code != http.StatusNoContent {
		t.Fatalf("status=%d", w.Code)
	}
}
