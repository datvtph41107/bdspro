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
	"github.com/datvtph41107/bdspro/routers/api/middleware"
	auth_service "github.com/datvtph41107/bdspro/services/auth"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type currentAccountStoreFake struct {
	db.Store
	account db.Account
	err error
	calls int
}

func (s *currentAccountStoreFake) GetAccountByID(_ context.Context, _ int64) (db.Account, error) {
	s.calls++
	return s.account, s.err
}

func TestMeReadsCurrentAccountAndDoesNotLeakDriverErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	maker, err := authcrypto.NewJWTMaker(strings.Repeat("x", 32))
	if err != nil {
		t.Fatal(err)
	}
	access, _, err := maker.CreateToken(42, time.Minute, authcrypto.TokenTypeAccess)
	if err != nil {
		t.Fatal(err)
	}
	store := &currentAccountStoreFake{account: db.Account{
		ID: 42, PersonID: 67, Email: "verified@example.test",
		IsEmailVerified: true, CreatedAt: time.Date(2026, 11, 4, 0, 0, 0, 0, time.UTC),
	}}
	h := NewAuthHandler(auth_service.NewAuthService(store, nil, setting.Config{}), nil)
	r := gin.New()
	r.GET("/users/me", middleware.Authenticate(maker), h.Me)

	get := func(bearer string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/users/me", nil)
		if bearer != "" {
			req.Header.Set("Authorization", "Bearer "+bearer)
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}

	if w := get(""); w.Code != http.StatusUnauthorized || store.calls != 0 {
		t.Fatalf("unverified request reached account lookup: status=%d calls=%d", w.Code, store.calls)
	}
	w := get(access)
	if w.Code != http.StatusOK {
		t.Fatalf("live account HTTP status=%d body=%s", w.Code, w.Body.String())
	}
	var payload struct {
		AccountID int64 `json:"account_id"`
		Email string `json:"email"`
		IsEmailVerified bool `json:"is_email_verified"`
		CreatedAt time.Time `json:"created_at"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.AccountID != 42 || payload.Email != "verified@example.test" || !payload.IsEmailVerified ||
		!payload.CreatedAt.Equal(store.account.CreatedAt) {
		t.Fatalf("unexpected current-account response: %+v", payload)
	}
	store.err = pgx.ErrNoRows
	w = get(access)
	if w.Code != http.StatusUnauthorized || !strings.Contains(w.Body.String(), "auth.account_unavailable") {
		t.Fatalf("deleted account must be rejected: status=%d body=%s", w.Code, w.Body.String())
	}
	store.err = errors.New("private-password=db-secret")
	w = get(access)
	if w.Code != http.StatusInternalServerError || strings.Contains(w.Body.String(), "db-secret") {
		t.Fatalf("unexpected internal response: status=%d body=%s", w.Code, w.Body.String())
	}
}
