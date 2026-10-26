package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	db "github.com/datvtph41107/bdspro/db/sqlc"
	"github.com/datvtph41107/bdspro/mail"
	"github.com/datvtph41107/bdspro/util"
	"github.com/datvtph41107/bdspro/worker"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

// Full channel test: HTTP -> PostgreSQL commit -> Redis/Asynq -> SMTP -> Mailpit -> HTTP verify.
// Only runs on disposable GitHub Actions infrastructure; never against the developer's DB.
func TestRegisterEmailEndToEnd(t *testing.T) {
	if os.Getenv("GITHUB_ACTIONS") != "true" {
		t.Skip("full registration test requires disposable CI PostgreSQL")
	}
	mailpitAPI := os.Getenv("MAILPIT_API_URL")
	smtpAddress := os.Getenv("MAILPIT_SMTP_ADDRESS")
	if mailpitAPI == "" || smtpAddress == "" {
		t.Skip("Mailpit SMTP and HTTP endpoints not configured")
	}
	config, err := util.LoadConfig("../../..")
	require.NoError(t, err)
	config.EmailSenderAddress = "no-reply@bdspro.test"

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, config.DBSource)
	require.NoError(t, err)
	defer pool.Close()
	require.NoError(t, pool.Ping(ctx))
	store := db.NewStore(pool)

	redisOpt := asynq.RedisClientOpt{Addr: "127.0.0.1:6379"}
	mailer := mail.NewSMTPSender("BDSPro", config.EmailSenderAddress, "", smtpAddress, false)
	processor := worker.NewRedisTaskProcessor(redisOpt, store, mailer, config.VerifyEmailURL)
	require.NoError(t, processor.Start())
	defer processor.Shutdown()
	distributor := worker.NewRedisTaskDistributor(redisOpt)
	defer distributor.Close()
	server, err := NewServer(config, store, distributor)
	require.NoError(t, err)

	emailAddress := uuid.NewString() + "@example.test"
	body, err := json.Marshal(map[string]string{
		"email": emailAddress,
		"password": "integration-password-123",
	})
	require.NoError(t, err)
	request := httptest.NewRequest(http.MethodPost, "/users", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	require.Equal(t, http.StatusCreated, response.Code)
	require.NotEmpty(t, response.Header().Get("X-Request-ID"))
	var result struct{ AccountID int64 `json:"account_id"` }
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &result))
	require.Positive(t, result.AccountID)

	var persistedEmail string
	var verified bool
	require.NoError(t, pool.QueryRow(ctx,
		"SELECT email, is_email_verified FROM accounts WHERE id=$1", result.AccountID,
	).Scan(&persistedEmail, &verified))
	require.Equal(t, emailAddress, persistedEmail)
	require.False(t, verified)

	// Mailpit stores messages received via real SMTP, not a mocked sender.
	var messageID string
	client := &http.Client{Timeout: 3 * time.Second}
	deadline := time.Now().Add(45 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := client.Get(mailpitAPI + "/api/v1/messages?limit=100")
		if err == nil {
			var inbox struct {
				Messages []struct {
					ID string `json:"ID"`
					To []struct{ Address string `json:"Address"` } `json:"To"`
				} `json:"messages"`
			}
			decodeErr := json.NewDecoder(resp.Body).Decode(&inbox)
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK && decodeErr == nil {
				for _, message := range inbox.Messages {
					for _, recipient := range message.To {
						if strings.EqualFold(recipient.Address, emailAddress) {
							messageID = message.ID
							}
					}
				}
			}
		}
		if messageID != "" { break }
		time.Sleep(500 * time.Millisecond)
	}
	require.NotEmpty(t, messageID, "Mailpit did not receive verification email")

	resp, err := client.Get(mailpitAPI + "/api/v1/message/" + url.PathEscape(messageID))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var message struct {
		HTML string `json:"HTML"`
		Subject string `json:"Subject"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&message))
	require.Equal(t, "Verify your BDSPro email", message.Subject)
	match := regexp.MustCompile(`href="([^"]+)"`).FindStringSubmatch(message.HTML)
	require.Len(t, match, 2, "verification link missing from received email")
	link, err := url.Parse(html.UnescapeString(match[1]))
	require.NoError(t, err)
	require.Equal(t, "/users/verify_email", link.Path)
	require.NotEmpty(t, link.Query().Get("email_id"))
	require.NotEmpty(t, link.Query().Get("secret_code"))

	verification := httptest.NewRecorder()
	server.Handler().ServeHTTP(verification,
		httptest.NewRequest(http.MethodGet, link.RequestURI(), nil))
	require.Equal(t, http.StatusOK, verification.Code)
	require.NoError(t, pool.QueryRow(ctx,
		"SELECT is_email_verified FROM accounts WHERE id=$1", result.AccountID,
	).Scan(&verified))
	require.True(t, verified)

	var used int
	require.NoError(t, pool.QueryRow(ctx,
		"SELECT count(*) FROM verify_emails WHERE account_id=$1 AND is_used=true", result.AccountID,
	).Scan(&used))
	require.Equal(t, 1, used)
	t.Log(fmt.Sprintf("register + SMTP + verification passed for account_id=%d", result.AccountID))
}
