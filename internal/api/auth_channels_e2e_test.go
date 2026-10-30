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
	"github.com/datvtph41107/bdspro/internal/jobs"
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
		TokenSymmetricKey:    strings.Repeat("x", 32),
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
	check, lookupErr := db.NewStore(pool).GetAccountProfile(ctx, int64(firstID.(float64)))
	if lookupErr != nil || !check.HasPhone || check.HasPassword || check.HasGoogle {
		t.Fatalf("phone signup profile mismatch: %+v, %v", check, lookupErr)
	}
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
	assertPasswordResetIntent(t, ctx, pool, account.ID)
	if err := jobs.SendPasswordResetEmail(ctx, store, mailer, "http://localhost:8080/users/verify_email", account.ID); err != nil {
		t.Fatal(err)
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

func TestEmailPasswordRegistrationAndDuplicateHTTPAcceptance(t *testing.T) {
	_, _, server, _ := authAcceptanceServer(t)
	email := uuid.NewString() + "@example.test"
	status, created := authAcceptanceCall(t, server, "/users", http.MethodPost, map[string]string{
		"email": email, "password": "InitialPassword123",
	}, "")
	if status != http.StatusCreated {
		t.Fatalf("email signup status=%d response=%v", status, created)
	}
	accountID := created["account_id"]
	status, rejected := authAcceptanceCall(t, server, "/users", http.MethodPost, map[string]string{
		"email": strings.ToUpper(email), "password": "InitialPassword123",
	}, "")
	if status != http.StatusConflict || rejected["code"] != "auth.email_registered" {
		t.Fatalf("duplicate email must conflict: status=%d response=%v", status, rejected)
	}
	status, login := authAcceptanceCall(t, server, "/users/login", http.MethodPost, map[string]string{
		"email": email, "password": "InitialPassword123",
	}, "")
	if status != http.StatusOK || login["user"].(map[string]any)["account_id"] != accountID {
		t.Fatalf("email login did not preserve AccountID: status=%d response=%v", status, login)
	}
}

func TestPhoneOnlyAccountAttachesEmailAndPasswordOnSameAccount(t *testing.T) {
	ctx, pool, server, mailer := authAcceptanceServer(t)
	phone := fmt.Sprintf("+84%09d", time.Now().UnixNano()%1000000000)
	email := uuid.NewString() + "@example.test"

	status, _ := authAcceptanceCall(t, server, "/users/phone/register/start", http.MethodPost, map[string]string{"phone": phone}, "")
	if status != http.StatusAccepted {
		t.Fatalf("phone signup start: %d", status)
	}
	var code string
	if err := pool.QueryRow(ctx, "SELECT dev_code FROM phone_signup_challenges WHERE phone_e164=$1", phone).Scan(&code); err != nil {
		t.Fatal(err)
	}
	status, registered := authAcceptanceCall(t, server, "/users/phone/register/verify", http.MethodPost, map[string]string{"phone": phone, "code": code}, "")
	if status != http.StatusCreated {
		t.Fatalf("phone signup: %d %v", status, registered)
	}
	id := registered["user"].(map[string]any)["account_id"]
	access := registered["access_token"].(string)

	status, _ = authAcceptanceCall(t, server, "/users/phone/login/start", http.MethodPost, map[string]string{"phone": phone}, "")
	if status != http.StatusAccepted {
		t.Fatalf("fresh OTP: %d", status)
	}
	if err := pool.QueryRow(ctx, "SELECT dev_code FROM phone_otp_challenges WHERE phone_e164=$1 AND purpose='login'", phone).Scan(&code); err != nil {
		t.Fatal(err)
	}
	status, result := authAcceptanceCall(t, server, "/users/me/email/start", http.MethodPost, map[string]string{"email": email, "phone_code": code}, access)
	if status != http.StatusAccepted {
		t.Fatalf("email link start: %d %v", status, result)
	}

	token := regexp.MustCompile(`<code>([A-Za-z0-9_-]{43})</code>`).FindStringSubmatch(mailer.body)
	if len(token) != 2 {
		t.Fatalf("email link token not mailed")
	}
	status, result = authAcceptanceCall(t, server, "/users/me/email/verify", http.MethodPost, map[string]string{"token": token[1]}, access)
	if status != http.StatusNoContent {
		t.Fatalf("email confirm: %d %v", status, result)
	}
	current, err := db.NewStore(pool).GetAccountByID(ctx, int64(id.(float64)))
	if err != nil || current.Email != email || !current.IsEmailVerified {
		t.Fatalf("email link mutated identity: %+v err=%v", current, err)
	}
	status, _ = authAcceptanceCall(t, server, "/users/me/email/verify", http.MethodPost, map[string]string{"token": token[1]}, access)
	if status != http.StatusBadRequest {
		t.Fatalf("email link replay status=%d", status)
	}

	status, _ = authAcceptanceCall(t, server, "/users/password/reset/request", http.MethodPost, map[string]string{"email": email}, "")
	if status != http.StatusAccepted {
		t.Fatalf("password setup request: %d", status)
	}
	assertPasswordResetIntent(t, ctx, pool, int64(id.(float64)))
	if err := jobs.SendPasswordResetEmail(ctx, db.NewStore(pool), mailer, "http://localhost:8080/users/verify_email", int64(id.(float64))); err != nil {
		t.Fatal(err)
	}
	reset := regexp.MustCompile(`token=([A-Za-z0-9_-]{43})`).FindStringSubmatch(mailer.body)
	if len(reset) != 2 {
		t.Fatal("password setup token not mailed")
	}
	status, result = authAcceptanceCall(t, server, "/users/password/reset", http.MethodPost, map[string]string{"token": reset[1], "password": "NewPhoneAccountPassword123"}, "")
	if status != http.StatusNoContent {
		t.Fatalf("password setup: %d %v", status, result)
	}
	status, result = authAcceptanceCall(t, server, "/users/login", http.MethodPost, map[string]string{"email": email, "password": "NewPhoneAccountPassword123"}, "")
	if status != http.StatusOK || result["user"].(map[string]any)["account_id"] != id {
		t.Fatalf("email login changed AccountID: %d %v", status, result)
	}
}

func TestSessionOwnershipRevocationAndRefreshDenial(t *testing.T) {
	_, _, server, _ := authAcceptanceServer(t)
	email := uuid.NewString() + "@example.test"
	_, _ = authAcceptanceCall(t, server, "/users", http.MethodPost, map[string]string{"email": email, "password": "AReallyStrongPassword123"}, "")
	status, login := authAcceptanceCall(t, server, "/users/login", http.MethodPost, map[string]string{"email": email, "password": "AReallyStrongPassword123"}, "")
	if status != http.StatusOK {
		t.Fatalf("first login: %d %v", status, login)
	}
	firstToken := login["access_token"].(string)
	firstSession := login["session_id"].(string)
	refresh := login["refresh_token"].(string)
	status, secondLogin := authAcceptanceCall(t, server, "/users/login", http.MethodPost, map[string]string{"email": email, "password": "AReallyStrongPassword123"}, "")
	if status != http.StatusOK {
		t.Fatalf("second login: %d %v", status, secondLogin)
	}
	status, list := authAcceptanceCall(t, server, "/users/me/sessions", http.MethodGet, nil, firstToken)
	if status != http.StatusOK || len(list["sessions"].([]any)) < 2 {
		t.Fatalf("list sessions status=%d result=%v", status, list)
	}
	for _, item := range list["sessions"].([]any) {
		session := item.(map[string]any)
		if _, ok := session["refresh_token"]; ok {
			t.Fatal("session listing leaked refresh token")
		}
	}
	otherEmail := uuid.NewString() + "@example.test"
	_, _ = authAcceptanceCall(t, server, "/users", http.MethodPost, map[string]string{"email": otherEmail, "password": "AReallyStrongPassword123"}, "")
	_, otherLogin := authAcceptanceCall(t, server, "/users/login", http.MethodPost, map[string]string{"email": otherEmail, "password": "AReallyStrongPassword123"}, "")
	otherToken := otherLogin["access_token"].(string)
	status, _ = authAcceptanceCall(t, server, "/users/me/sessions/"+firstSession, http.MethodDelete, nil, otherToken)
	if status != http.StatusNotFound {
		t.Fatalf("other account revoked session: %d", status)
	}
	status, _ = authAcceptanceCall(t, server, "/users/me/sessions/"+firstSession, http.MethodDelete, nil, firstToken)
	if status != http.StatusNoContent {
		t.Fatalf("own revoke: %d", status)
	}
	status, _ = authAcceptanceCall(t, server, "/tokens/renew_access", http.MethodPost, map[string]string{"refresh_token": refresh}, "")
	if status != http.StatusUnauthorized {
		t.Fatalf("revoked refresh still renewed: %d", status)
	}
	status, _ = authAcceptanceCall(t, server, "/users/me/sessions/revoke_all", http.MethodPost, nil, firstToken)
	if status != http.StatusNoContent {
		t.Fatalf("revoke all: %d", status)
	}
	status, list = authAcceptanceCall(t, server, "/users/me/sessions", http.MethodGet, nil, firstToken)
	if status != http.StatusOK || len(list["sessions"].([]any)) != 0 {
		t.Fatalf("revoke all failed: %d %v", status, list)
	}
}

// assertPasswordResetIntent confirms HTTP persists the request but does not
// persist the plaintext token or invoke SMTP synchronously.
func assertPasswordResetIntent(t *testing.T, ctx context.Context, pool *pgxpool.Pool, accountID int64) {
	t.Helper()
	var taskType string
	var payload []byte
	err := pool.QueryRow(ctx, `SELECT task_type, payload FROM job_outbox
        WHERE (payload->>'account_id')::bigint=$1 AND task_type='task:send_password_reset'
        ORDER BY id DESC LIMIT 1`, accountID).Scan(&taskType, &payload)
	if err != nil {
		t.Fatal(err)
	}
	if taskType != "task:send_password_reset" {
		t.Fatalf("unexpected job type: %s", taskType)
	}
	var data map[string]any
	if err := json.Unmarshal(payload, &data); err != nil {
		t.Fatal(err)
	}
	if len(data) != 1 || data["account_id"] != float64(accountID) {
		t.Fatalf("reset intent must carry only AccountID: %v", data)
	}
}
