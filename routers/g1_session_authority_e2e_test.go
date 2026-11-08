package routers

import (
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"strings"
	"testing"

	db "github.com/datvtph41107/bdspro/models/db"
	authcrypto "github.com/datvtph41107/bdspro/modules/auth"
	"github.com/google/uuid"
)

const g12Password = "InitialG12Password123"

func g12RegisterAndLogin(t *testing.T, server *Server) (string, map[string]any) {
	t.Helper()
	email := uuid.NewString() + "@example.test"
	status, created := authAcceptanceCall(t, server, "/users", http.MethodPost,
		map[string]string{"email": email, "password": g12Password}, "")
	if status != http.StatusCreated {
		t.Fatalf("register account: status=%d body=%v", status, created)
	}
	status, login := authAcceptanceCall(t, server, "/users/login", http.MethodPost,
		map[string]string{"email": email, "password": g12Password}, "")
	if status != http.StatusOK {
		t.Fatalf("login account: status=%d body=%v", status, login)
	}
	return email, login
}

func g12RequireHTTPStatus(t *testing.T, server *Server, path, method string, request any, bearer string, want int) map[string]any {
	t.Helper()
	status, body := authAcceptanceCall(t, server, path, method, request, bearer)
	if status != want {
		t.Fatalf("%s %s status=%d want=%d response=%v", method, path, status, want, body)
	}
	return body
}

func TestG12SessionRevokeDeniesRefreshButNotExistingAccessHTTP(t *testing.T) {
	_, pool, server, _ := authAcceptanceServer(t)
	email, first := g12RegisterAndLogin(t, server)
	_, unrelated := g12RegisterAndLogin(t, server)
	second := g12RequireHTTPStatus(t, server, "/users/login", http.MethodPost,
		map[string]string{"email": email, "password": g12Password}, "", http.StatusOK)
	firstAccess := first["access_token"].(string)
	firstRefresh := first["refresh_token"].(string)
	secondRefresh := second["refresh_token"].(string)
	secondAccess := second["access_token"].(string)
	otherRefresh := unrelated["refresh_token"].(string)

	// The current protocol does bind signed first-party access JWTs to a
	// session ID; ordinary Gin Authenticate simply does not resolve it.
	maker, err := authcrypto.NewJWTMaker(strings.Repeat("x", 32))
	if err != nil {
		t.Fatal(err)
	}
	firstClaims, err := maker.VerifyToken(firstAccess, authcrypto.TokenTypeAccess)
	if err != nil {
		t.Fatal(err)
	}
	firstSessionID, err := uuid.Parse(first["session_id"].(string))
	if err != nil || firstClaims.SessionID != firstSessionID {
		t.Fatalf("access JWT lost session binding: claims=%+v id=%s err=%v", firstClaims, first["session_id"], err)
	}
	// One-session revoke changes PostgreSQL authority for refresh only.
	g12RequireHTTPStatus(t, server, "/tokens/revoke", http.MethodPost,
		map[string]string{"refresh_token": firstRefresh}, "", http.StatusNoContent)
	var blocked bool
	if err := pool.QueryRow(t.Context(), "SELECT is_blocked FROM sessions WHERE id=$1", firstSessionID).Scan(&blocked); err != nil || !blocked {
		t.Fatalf("revoke did not persist: blocked=%t err=%v", blocked, err)
	}
	g12RequireHTTPStatus(t, server, "/tokens/renew_access", http.MethodPost,
		map[string]string{"refresh_token": firstRefresh}, "", http.StatusUnauthorized)
	g12RequireHTTPStatus(t, server, "/users/me", http.MethodGet, nil, firstAccess, http.StatusOK)
	// An access token whose refresh session was revoked can still MUTATE the
	// profile under today's stateless route policy, not merely read it.
	g12RequireHTTPStatus(t, server, "/users/me/profile", http.MethodPatch,
		map[string]string{"display_name": "Changed by revoked session"}, firstAccess, http.StatusOK)
	profile := g12RequireHTTPStatus(t, server, "/users/me", http.MethodGet, nil, secondAccess, http.StatusOK)
	if profile["profile"].(map[string]any)["display_name"] != "Changed by revoked session" {
		t.Fatalf("revoked JWT profile mutation not persisted: %+v", profile)
	}
	g12RequireHTTPStatus(t, server, "/tokens/renew_access", http.MethodPost,
		map[string]string{"refresh_token": secondRefresh}, "", http.StatusOK)
	g12RequireHTTPStatus(t, server, "/tokens/renew_access", http.MethodPost,
		map[string]string{"refresh_token": otherRefresh}, "", http.StatusOK)

	// Security implication of the current stateless contract: the old access
	// JWT can still invoke an account-wide session revocation operation.
	g12RequireHTTPStatus(t, server, "/users/me/sessions/revoke_all", http.MethodPost,
		nil, firstAccess, http.StatusNoContent)
	g12RequireHTTPStatus(t, server, "/tokens/renew_access", http.MethodPost,
		map[string]string{"refresh_token": secondRefresh}, "", http.StatusUnauthorized)
	g12RequireHTTPStatus(t, server, "/users/me", http.MethodGet, nil, firstAccess, http.StatusOK)
	g12RequireHTTPStatus(t, server, "/users/me", http.MethodGet, nil, secondAccess, http.StatusOK)
	g12RequireHTTPStatus(t, server, "/tokens/renew_access", http.MethodPost,
		map[string]string{"refresh_token": otherRefresh}, "", http.StatusOK)
}

func TestG12PasswordRecoveryRevokesRefreshNotIssuedAccessHTTP(t *testing.T) {
	ctx, pool, server, _ := authAcceptanceServer(t)
	email, login := g12RegisterAndLogin(t, server)
	accountID := int64(login["user"].(map[string]any)["account_id"].(float64))
	store := db.NewStore(pool)
	if _, err := store.MarkEmailVerified(ctx, db.MarkEmailVerifiedParams{ID: accountID, Email: email}); err != nil {
		t.Fatal(err)
	}
	access := login["access_token"].(string)
	refresh := login["refresh_token"].(string)
	// Recovery proof is created only for a disposable test account; no SMTP
	// message or external identifier is involved.
	proof := base64.RawURLEncoding.EncodeToString([]byte(strings.Repeat("p", 32)))
	digest := sha256.Sum256([]byte(proof))
	if err := store.CreatePasswordResetToken(ctx, db.CreatePasswordResetTokenParams{
		AccountID: accountID, TokenHash: digest[:],
	}); err != nil {
		t.Fatal(err)
	}
	const replacement = "NewG12Password456"
	g12RequireHTTPStatus(t, server, "/users/password/reset", http.MethodPost,
		map[string]string{"token": proof, "password": replacement}, "", http.StatusNoContent)
	g12RequireHTTPStatus(t, server, "/tokens/renew_access", http.MethodPost,
		map[string]string{"refresh_token": refresh}, "", http.StatusUnauthorized)
	g12RequireHTTPStatus(t, server, "/users/me", http.MethodGet, nil, access, http.StatusOK)
	g12RequireHTTPStatus(t, server, "/users/me/settings", http.MethodGet, nil, access, http.StatusOK)
	g12RequireHTTPStatus(t, server, "/users/login", http.MethodPost,
		map[string]string{"email": email, "password": g12Password}, "", http.StatusUnauthorized)
	fresh := g12RequireHTTPStatus(t, server, "/users/login", http.MethodPost,
		map[string]string{"email": email, "password": replacement}, "", http.StatusOK)
	g12RequireHTTPStatus(t, server, "/users/me", http.MethodGet, nil,
		fresh["access_token"].(string), http.StatusOK)
}
