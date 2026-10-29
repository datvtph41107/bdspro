package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/datvtph41107/bdspro/internal/config"
	db "github.com/datvtph41107/bdspro/internal/database"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type authAcceptanceMail struct {
	body string
}

func (m *authAcceptanceMail) SendEmail(_ string, content string, _, _, _, _ []string) error {
	m.body = content
	return nil
}

func authAcceptanceServer(t *testing.T) (context.Context, *pgxpool.Pool, *Server, *authAcceptanceMail) {
	t.Helper()
	url := os.Getenv("BDS_TEST_DB_URL")
	if url == "" {
		t.Skip("requires disposable test database")
	}
	if os.Getenv("BDS_TEST_MODE") != "disposable" {
		t.Fatal("refusing non-disposable database")
	}
	pgConfig, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	if pgConfig.ConnConfig.Database != "bdspro_test" {
		t.Fatal("refusing application database")
	}
	ctx := context.Background()
	pool, err := pgxpool.NewWithConfig(ctx, pgConfig)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if err := pool.Ping(ctx); err != nil {
		t.Fatal(err)
	}
	redisClient := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379"})
	t.Cleanup(func() { _ = redisClient.Close() })
	if err := redisClient.Ping(ctx).Err(); err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{
		Environment:          "development",
		TokenSymmetricKey:     strings.Repeat("x", 32),
		AccessTokenDuration:  15 * time.Minute,
		RefreshTokenDuration: 24 * time.Hour,
		VerifyEmailURL:       "http://localhost:8080/users/verify_email",
	}
	mailer := &authAcceptanceMail{}
	server, err := NewServer(cfg, db.NewStore(pool), redisClient, mailer)
	if err != nil {
		t.Fatal(err)
	}
	gin.SetMode(gin.TestMode)
	return ctx, pool, server, mailer
}

func authAcceptanceCall(t *testing.T, server *Server, path, method string, request any, bearer string) (int, map[string]any) {
	t.Helper()
	body, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(method, path, bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	if bearer != "" {
		r.Header.Set("Authorization", "Bearer "+bearer)
	}
	w := httptest.NewRecorder()
	server.Handler().ServeHTTP(w, r)
	var response map[string]any
	if w.Body.Len() > 0 {
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("not a JSON response: status=%d body=%s error=%v", w.Code, w.Body.String(), err)
		}
	}
	return w.Code, response
}

func TestPhoneSignupLoginAndBusinessHTTPAcceptance(t *testing.T) {
	ctx, pool, server, _ := authAcceptanceServer(t)
	phone := fmt.Sprintf("+84%09d", time.Now().UnixNano()%1000000000)
	status, _ := authAcceptanceCall(t, server, "/users/phone/register/start", http.MethodPost, map[string]string{"phone": phone}, "")
	if status != http.StatusAccepted {
		t.Fatalf("signup start status=%d", status)
	}
	var code string
	if err := pool.QueryRow(ctx, "SELECT dev_code FROM phone_signup_challenges WHERE phone_e164=$1", phone).Scan(&code); err != nil {
		t.Fatal(err)
	}
	status, result := authAcceptanceCall(t, server, "/users/phone/register/verify", http.MethodPost, map[string]string{"phone": phone, "code": code}, "")
	if status != http.StatusCreated {
		t.Fatalf("signup verify status=%d result=%v", status, result)
	}
	firstID := result["user"].(map[string]any)["account_id"]
	access := result["access_token"].(string)
	status, result = authAcceptanceCall(t, server, "/organizations", http.MethodPost, map[string]string{"name": "Phone Business"}, access)
	if status != http.StatusCreated {
		t.Fatalf("verified phone cannot create organization: %d %v", status, result)
	}
	status, result = authAcceptanceCall(t, server, "/users/phone/register/start", http.MethodPost, map[string]string{"phone": phone}, "")
	if status != http.StatusConflict || result["code"] != "auth.phone_already_registered" {
		t.Fatalf("duplicate phone must conflict: status=%d result=%v", status, result)
	}
	status, _ = authAcceptanceCall(t, server, "/users/phone/login/start", http.MethodPost, map[string]string{"phone": phone}, "")
	if status != http.StatusAccepted {
		t.Fatalf("login start status=%d", status)
	}
	if err := pool.QueryRow(ctx, "SELECT dev_code FROM phone_otp_challenges WHERE phone_e164=$1 AND purpose='login'", phone).Scan(&code); err != nil {
		t.Fatal(err)
	}
	status, result = authAcceptanceCall(t, server, "/users/phone/login/verify", http.MethodPost, map[string]string{"phone": phone, "code": code}, "")
	if status != http.StatusOK || result["user"].(map[string]any)["account_id"] != firstID {
		t.Fatalf("OTP login changed identity: status=%d result=%v", status, result)
	}
	status, _ = authAcceptanceCall(t, server, "/users/phone/login/verify", http.MethodPost, map[string]string{"phone": phone, "code": code}, "")
	if status != http.StatusUnauthorized {
		t.Fatalf("consumed OTP was replayed: status=%d", status)
	}
	// Unknown login phone does not disclose whether a credential exists.
	unknown := fmt.Sprintf("+84%09d", (time.Now().UnixNano()+1234567)%1000000000)
	status, _ = authAcceptanceCall(t, server, "/users/phone/login/start", http.MethodPost, map[string]string{"phone": unknown}, "")
	if status != http.StatusAccepted {
		t.Fatalf("unknown phone disclosed registration: status=%d", status)
	}
}

func TestPasswordResetAndGoogleCreatedAccountHTTPAcceptance(t *testing.T) {
	ctx, pool, server, mailer := authAcceptanceServer(t)
	store := db.NewStore(pool)
	email := uuid.NewString() + "@example.test"
	account, err := store.CreateGoogleAccountTx(ctx, email, uuid.NewString())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetPasswordHash(ctx, account.ID); err != pgx.ErrNoRows {
		t.Fatalf("Google account unexpectedly has a password: %v", err)
	}
	status, _ := authAcceptanceCall(t, server, "/users/password/reset/request", http.MethodPost, map[string]string{"email": email}, "")
	if status != http.StatusAccepted {
		t.Fatalf("reset request status=%d", status)
	}
	match := regexp.MustCompile(`token=([A-Za-z0-9_-]{43})`).FindStringSubmatch(mailer.body)
	if len(match) != 2 {
		t.Fatal("reset email did not contain one-time token")
	}
	status, result := authAcceptanceCall(t, server, "/users/password/reset", http.MethodPost, map[string]string{
		"token": match[1], "password": "TestNewPassword123",
	}, "")
	if status != http.StatusNoContent {
		t.Fatalf("password setup status=%d response=%v", status, result)
	}
	status, result = authAcceptanceCall(t, server, "/users/login", http.MethodPost, map[string]string{
		"email": email, "password": "TestNewPassword123",
	}, "")
	if status != http.StatusOK || result["user"].(map[string]any)["account_id"] != float64(account.ID) {
		t.Fatalf("password sign-in did not preserve Google AccountID: status=%d result=%v", status, result)
	}
	status, _ = authAcceptanceCall(t, server, "/users/password/reset", http.MethodPost, map[string]string{
		"token": match[1], "password": "TestSecondPassword123",
	}, "")
	if status != http.StatusBadRequest {
		t.Fatalf("used reset token was accepted: status=%d", status)
	}
}
