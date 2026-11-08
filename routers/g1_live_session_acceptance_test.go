package routers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	authcrypto "github.com/datvtph41107/bdspro/modules/auth"
	appctx "github.com/datvtph41107/bdspro/services/context"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func TestG12LiveSessionRejectsLegacyAndCrossAccountJWT(t *testing.T) {
	_, _, server, _ := authAcceptanceServer(t)
	_, account := g12RegisterAndLogin(t, server)
	_, unrelated := g12RegisterAndLogin(t, server)
	maker, err := authcrypto.NewJWTMaker(strings.Repeat("x", 32))
	if err != nil {
		t.Fatal(err)
	}
	id := int64(account["user"].(map[string]any)["account_id"].(float64))
	legacy, _, err := maker.CreateToken(id, time.Minute, authcrypto.TokenTypeAccess)
	if err != nil {
		t.Fatal(err)
	}
	g12RequireHTTPStatus(t, server, "/users/me", http.MethodGet, nil, legacy, http.StatusUnauthorized)

	unrelatedID, err := uuid.Parse(unrelated["session_id"].(string))
	if err != nil {
		t.Fatal(err)
	}
	forgedAuthority, _, err := maker.CreateSessionAccessToken(id, time.Minute, unrelatedID)
	if err != nil {
		t.Fatal(err)
	}
	g12RequireHTTPStatus(t, server, "/users/me", http.MethodGet, nil, forgedAuthority, http.StatusUnauthorized)
	g12RequireHTTPStatus(t, server, "/users/me", http.MethodGet, nil, account["access_token"].(string), http.StatusOK)
}

func TestG12LiveSessionPostgresFailureFailsClosedHTTP(t *testing.T) {
	_, pool, server, _ := authAcceptanceServer(t)
	_, login := g12RegisterAndLogin(t, server)
	pool.Close() // This server uses a disposable per-test pool.
	status, body := authAcceptanceCall(t, server, "/users/me", http.MethodGet, nil, login["access_token"].(string))
	if status != http.StatusServiceUnavailable || body["code"] != "identity_store_unavailable" {
		t.Fatalf("Postgres failure permitted a protected request: status=%d body=%v", status, body)
	}
}

func TestG12InFlightRequestMayFinishAfterRevokeButNewRequestCannotStart(t *testing.T) {
	_, _, server, _ := authAcceptanceServer(t)
	_, login := g12RegisterAndLogin(t, server)
	access := login["access_token"].(string)
	refresh := login["refresh_token"].(string)
	entered := make(chan struct{})
	release := make(chan struct{})
	finished := make(chan int, 1)
	defer func() {
		select {
		case <-release:
		default:
			close(release)
		}
	}()
	server.router.GET("/__g12_inflight_test",
		appctx.Authenticate(server.tokenMaker),
		appctx.RequireSession(server.store),
		func(c *gin.Context) {
			close(entered)
			<-release
			c.Status(http.StatusNoContent)
		},
	)
	request := httptest.NewRequest(http.MethodGet, "/__g12_inflight_test", nil)
	request.Header.Set("Authorization", "Bearer "+access)
	go func() {
		w := httptest.NewRecorder()
		server.Handler().ServeHTTP(w, request)
		finished <- w.Code
	}()
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("request did not reach post-auth handler")
	}
	g12RequireHTTPStatus(t, server, "/tokens/revoke", http.MethodPost,
		map[string]string{"refresh_token": refresh}, "", http.StatusNoContent)
	close(release)
	select {
	case status := <-finished:
		if status != http.StatusNoContent {
			t.Fatalf("pre-admitted request rejected after revoke: %d", status)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("pre-admitted handler failed to complete")
	}
	// New requests must check the session again; the middleware does not
	// promise it can cancel operations which crossed the gate before revoke.
	g12RequireHTTPStatus(t, server, "/__g12_inflight_test", http.MethodGet,
		nil, access, http.StatusUnauthorized)
}
