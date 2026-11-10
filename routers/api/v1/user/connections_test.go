package user

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	db "github.com/datvtph41107/bdspro/models/db"
	authcrypto "github.com/datvtph41107/bdspro/modules/auth"
	"github.com/datvtph41107/bdspro/modules/setting"
	"github.com/datvtph41107/bdspro/modules/web"
	auth_service "github.com/datvtph41107/bdspro/services/auth"
	appctx "github.com/datvtph41107/bdspro/services/context"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type connectionsStoreFake struct {
	db.Store
	accountID int64
	sessionID uuid.UUID
	calls     []int64
	err       error
}

func (s *connectionsStoreFake) GetSession(_ context.Context, id uuid.UUID) (db.Session, error) {
	if id != s.sessionID {
		return db.Session{}, pgx.ErrNoRows
	}
	return db.Session{ID: id, AccountID: s.accountID, ExpiresAt: time.Now().Add(time.Hour)}, nil
}

func (s *connectionsStoreFake) ListAccountConnectionHistories(_ context.Context, accountID int64) ([]db.ListAccountConnectionHistoriesRow, error) {
	s.calls = append(s.calls, accountID)
	if s.err != nil {
		return nil, s.err
	}
	if accountID != s.accountID {
		return nil, nil
	}
	return []db.ListAccountConnectionHistoriesRow{{
		ID: 10, SessionID: uuid.New(), Action: "login",
		IpAddress: "203.0.113.10", UserAgent: "TestClient",
		CreatedAt: time.Date(2026, 11, 5, 0, 0, 0, 0, time.UTC),
	}}, nil
}

func TestGetConnectionsUsesVerifiedAccountAndSafeErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	maker, err := authcrypto.NewJWTMaker(strings.Repeat("k", 32))
	if err != nil {
		t.Fatal(err)
	}
	sessionID := uuid.New()
	access, _, err := maker.CreateSessionAccessToken(42, time.Minute, sessionID)
	if err != nil {
		t.Fatal(err)
	}
	store := &connectionsStoreFake{accountID: 42, sessionID: sessionID}
	service := auth_service.NewAuthService(store, maker, setting.Config{}, nil)
	router := web.NewRouter()
	router.AfterRouting(appctx.APIContexter())
	engine := router.GinEngine()
	engine.Use(func(c *gin.Context) {
		ctx := db.WithStore(c.Request.Context(), store)
		ctx = authcrypto.WithMaker(ctx, maker)
		ctx = auth_service.WithService(ctx, service)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
	router.Get("/users/me/connections", appctx.Authenticate(), appctx.RequireSession(), GetConnections)
	request := func(bearer string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/users/me/connections?account_id=777", nil)
		if bearer != "" {
			req.Header.Set("Authorization", "Bearer "+bearer)
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w
	}
	if w := request(""); w.Code != http.StatusUnauthorized || len(store.calls) != 0 {
		t.Fatalf("unauthenticated request reached history: %d %+v", w.Code, store.calls)
	}
	w := request(access)
	if w.Code != http.StatusOK || len(store.calls) != 1 || store.calls[0] != 42 {
		t.Fatalf("untrusted account filter: %d %s %+v", w.Code, w.Body.String(), store.calls)
	}
	var payload map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	entries, ok := payload["connections"].([]any)
	if !ok || len(entries) != 1 || entries[0].(map[string]any)["action"] != "login" {
		t.Fatalf("unexpected history response: %s", w.Body.String())
	}
	connection := entries[0].(map[string]any)
	if _, ok := connection["session_id"].(string); !ok {
		t.Fatalf("connection session_id must be serialized as stable JSON string: %s", w.Body.String())
	}
	for _, forbidden := range []string{"account_id", "refresh_token"} {
		if _, exists := connection[forbidden]; exists {
			t.Fatalf("connection leaks field %q: %s", forbidden, w.Body.String())
		}
	}
	store.err = errors.New("sensitive sql details")
	w = request(access)
	if w.Code != http.StatusInternalServerError || strings.Contains(w.Body.String(), "sensitive sql details") {
		t.Fatalf("driver error leaked: %d %s", w.Code, w.Body.String())
	}
}
