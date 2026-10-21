package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/datvtph41107/bdspro/api"
	"github.com/datvtph41107/bdspro/auth"
	"github.com/datvtph41107/bdspro/token"
	"github.com/jackc/pgx/v5/pgxpool"
)

const sessionTestPassword = "ComplexPassphrase123!"

func registerSessionAccount(t *testing.T, db *pgxpool.Pool) (*auth.Service, int64, string) {
	t.Helper()
	s := auth.New(db, nil)
	id, firstToken := coreUser(t, s, "session-owner@example.test")
	return s, id, firstToken
}

func TestSessionCapKeepsFiveNewestActiveTokens(t *testing.T) {
	db := coreDB(t)
	s, accountID, first := registerSessionAccount(t, db)
	ctx := context.Background()
	tokens := []string{first}
	for i := 0; i < 6; i++ {
		result, err := s.Login(ctx, "session-owner@example.test", sessionTestPassword)
		if err != nil {
			t.Fatal(err)
		}
		tokens = append(tokens, result.Token)
	}
	var count int
	if err := db.QueryRow(ctx,
		"SELECT count(*) FROM sessions WHERE account_id=$1 AND revoked_at IS NULL AND expires_at>now()",
		accountID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 5 {
		t.Fatalf("active sessions = %d; want 5", count)
	}
	for i, token := range tokens {
		id, err := s.AccountFromToken(ctx, token)
		if i < len(tokens)-5 {
			if !errors.Is(err, auth.ErrUnauthorized) {
				t.Fatalf("evicted session %d still valid: %d, %v", i, id, err)
			}
		} else if err != nil || id != accountID {
			t.Fatalf("new session %d unexpectedly invalid: %d, %v", i, id, err)
		}
	}
	var total int
	if err := db.QueryRow(ctx, "SELECT count(*) FROM sessions WHERE account_id=$1", accountID).Scan(&total); err != nil {
		t.Fatal(err)
	}
	if total != len(tokens) {
		t.Fatalf("revocation erased session history: %d rows, expected %d", total, len(tokens))
	}
}

func TestSessionCapConcurrentLoginUsesOnePostgresLock(t *testing.T) {
	db := coreDB(t)
	s, accountID, _ := registerSessionAccount(t, db)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	const concurrency = 12
	results := make(chan error, concurrency)
	var workers sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			result, err := s.Login(ctx, "session-owner@example.test", sessionTestPassword)
			if err == nil && result.AccountID != accountID {
				err = fmt.Errorf("wrong session account: %d", result.AccountID)
			}
			results <- err
		}()
	}
	workers.Wait()
	close(results)
	for err := range results {
		if err != nil {
			t.Fatalf("concurrent login failed: %v", err)
		}
	}
	var active int
	if err := db.QueryRow(ctx,
		"SELECT count(*) FROM sessions WHERE account_id=$1 AND revoked_at IS NULL AND expires_at>now()",
		accountID).Scan(&active); err != nil {
		t.Fatal(err)
	}
	if active != 5 {
		t.Fatalf("concurrent logins bypassed cap: %d active sessions", active)
	}
}

func TestLogoutAllRevokesEverySessionButPermitsFutureLogin(t *testing.T) {
	db := coreDB(t)
	s, accountID, token := registerSessionAccount(t, db)
	ctx := context.Background()
	var tokens = []string{token}
	for i := 0; i < 2; i++ {
		newSession, err := s.Login(ctx, "session-owner@example.test", sessionTestPassword)
		if err != nil {
			t.Fatal(err)
		}
		tokens = append(tokens, newSession.Token)
	}
	server := api.NewServer(db, s, nil, nil, true, "")
	req := httptest.NewRequest(http.MethodPost, "/auth/logout-all", nil)
	req.AddCookie(&http.Cookie{Name: "bdspro_session", Value: tokens[0]})
	req.Header.Set("X-BDSPro-Request", "1")
	w := httptest.NewRecorder()
	server.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusNoContent || w.Body.Len() != 0 {
		t.Fatalf("logout-all status %d: %s", w.Code, w.Body.String())
	}
	if len(w.Result().Cookies()) != 1 || w.Result().Cookies()[0].MaxAge >= 0 {
		t.Fatal("logout-all must clear the caller cookie")
	}
	for i, token := range tokens {
		if _, err := s.AccountFromToken(ctx, token); !errors.Is(err, auth.ErrUnauthorized) {
			t.Fatalf("old session %d remains valid: %v", i, err)
		}
	}
	if err := s.LogoutAll(ctx, tokens[0]); !errors.Is(err, auth.ErrUnauthorized) {
		t.Fatalf("replaying logout-all must fail: %v", err)
	}
	result, err := s.Login(ctx, "session-owner@example.test", sessionTestPassword)
	if err != nil {
		t.Fatal(err)
	}
	id, err := s.AccountFromToken(ctx, result.Token)
	if err != nil || id != accountID {
		t.Fatalf("fresh login after logout-all: %d, %v", id, err)
	}
}

func TestLogoutAllWaitsForAccountLock(t *testing.T) {
	db := coreDB(t)
	s, id, token := registerSessionAccount(t, db)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	tx, err := db.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(context.Background())
	var lockedID int64
	if err := tx.QueryRow(ctx,
		"SELECT id FROM accounts WHERE id=$1 FOR UPDATE", id).Scan(&lockedID); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- s.LogoutAll(ctx, token) }()
	select {
	case err := <-done:
		t.Fatalf("logout-all bypassed the account row lock: %v", err)
	case <-time.After(75*time.Millisecond):
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("logout-all failed after lock released: %v", err)
		}
	case <-ctx.Done():
		t.Fatalf("logout-all stalled: %v", ctx.Err())
	}
	if _, err := s.AccountFromToken(ctx, token); !errors.Is(err, auth.ErrUnauthorized) {
		t.Fatalf("logout-all committed but token remains active: %v", err)
	}
}

func TestBearerLogoutAndLogoutAllRevokeRealPostgresSessions(t *testing.T) {
	db := coreDB(t)
	service, accountID, firstToken := registerSessionAccount(t, db)
	ctx := context.Background()
	issuer, err := token.NewJWTMaker(strings.Repeat("J", 32))
	if err != nil {
		t.Fatal(err)
	}
	server := api.NewServer(db, service, nil, nil, true, "")
	server.EnableJWTMaker(issuer)

	firstJWT, _, err := issuer.Issue(accountID, firstToken)
	if err != nil {
		t.Fatal(err)
	}
	logout := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	logout.Header.Set("Authorization", "Bearer "+firstJWT)
	logout.Header.Set("X-BDSPro-Request", "1")
	w := httptest.NewRecorder()
	server.Handler().ServeHTTP(w, logout)
	if w.Code != http.StatusNoContent {
		t.Fatalf("bearer logout status=%d body=%s", w.Code, w.Body.String())
	}
	if _, err := service.AccountFromToken(ctx, firstToken); !errors.Is(err, auth.ErrUnauthorized) {
		t.Fatalf("bearer logout left backing session live: %v", err)
	}
	check := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	check.Header.Set("Authorization", "Bearer "+firstJWT)
	w = httptest.NewRecorder()
	server.Handler().ServeHTTP(w, check)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("revoked signed access JWT still authenticates: %d", w.Code)
	}

	second, err := service.Login(ctx, "session-owner@example.test", sessionTestPassword)
	if err != nil {
		t.Fatal(err)
	}
	third, err := service.Login(ctx, "session-owner@example.test", sessionTestPassword)
	if err != nil {
		t.Fatal(err)
	}
	secondJWT, _, err := issuer.Issue(accountID, second.Token)
	if err != nil {
		t.Fatal(err)
	}
	all := httptest.NewRequest(http.MethodPost, "/auth/logout-all", nil)
	all.Header.Set("Authorization", "Bearer "+secondJWT)
	w = httptest.NewRecorder()
	server.Handler().ServeHTTP(w, all)
	if w.Code != http.StatusNoContent {
		t.Fatalf("bearer logout-all status=%d body=%s", w.Code, w.Body.String())
	}
	for _, token := range []string{second.Token, third.Token} {
		if _, err := service.AccountFromToken(ctx, token); !errors.Is(err, auth.ErrUnauthorized) {
			t.Fatalf("bearer logout-all left session active: %v", err)
		}
	}
	w = httptest.NewRecorder()
	server.Handler().ServeHTTP(w, all)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("replayed revoked bearer logout-all status=%d", w.Code)
	}
}
