package worker

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	account_model "github.com/datvtph41107/bdspro/models/account"
	auth_model "github.com/datvtph41107/bdspro/models/auth"
	db "github.com/datvtph41107/bdspro/models/db"
	"github.com/datvtph41107/bdspro/modules/queue"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"
)

// g1DurableMailSink stands in for an SMTP server that accepted delivery.
// The first child deliberately stops AFTER this side effect but BEFORE the
// Asynq handler can acknowledge its task. It is NOT a live SMTP protocol test.
type g1DurableMailSink struct {
	path  string
	block bool
}

func (s *g1DurableMailSink) SendEmail(_ string, body string, _, _, _, _ []string) error {
	f, err := os.OpenFile(s.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintln(f, body); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if s.block {
		// The parent kills this process. No handler acknowledgement occurs.
		select {}
	}
	return nil
}

// TestG1CrashWorkerChild runs only as an explicitly launched test subprocess.
// It uses the production Asynq processor and actual password-reset handler.
func TestG1CrashWorkerChild(t *testing.T) {
	role := os.Getenv("BDS_G1_CHILD_ROLE")
	if role == "" {
		t.Skip("subprocess-only test helper")
	}
	if role != "blocked" && role != "recovered" {
		t.Fatal("invalid child role")
	}
	if os.Getenv("BDS_TEST_MODE") != "disposable" {
		t.Fatal("worker child requires disposable mode")
	}
	source := os.Getenv("BDS_TEST_DB_URL")
	cfg, err := pgxpool.ParseConfig(source)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ConnConfig.Database != "bdspro_test" {
		t.Fatal("refusing non-test PostgreSQL")
	}
	ctx := context.Background()
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		t.Fatal(err)
	}
	q := os.Getenv("BDS_G1_QUEUE")
	if q == "" || os.Getenv("BDS_G1_MAIL_FILE") == "" {
		t.Fatal("missing isolated queue or mail sink")
	}
	opts := asynq.RedisClientOpt{Addr: os.Getenv("BDS_TEST_REDIS_ADDR"), DB: 15}
	proc, err := queue.NewRedisTaskProcessorWithConfig(opts, queue.Config{
		Queues: map[string]int{q: 1}, Concurrency: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	mailer := &g1DurableMailSink{
		path: os.Getenv("BDS_G1_MAIL_FILE"), block: role == "blocked",
	}
	proc.Handle(TaskSendPasswordReset, NewPasswordResetHandler(db.NewStore(pool), mailer, "https://example.test"))
	if err := proc.Start(); err != nil {
		t.Fatal(err)
	}
	shutdown, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-shutdown.Done()
	proc.Shutdown()
}

func g1LaunchChild(t *testing.T, role, queueName, mailFile string) (*exec.Cmd, func()) {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(executable, "-test.run=^TestG1CrashWorkerChild$", "-test.timeout=4m")
	logFile, err := os.CreateTemp(t.TempDir(), "worker-child-*.log")
	if err != nil {
		t.Fatal(err)
	}
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.Env = append(os.Environ(),
		"BDS_G1_CHILD_ROLE="+role,
		"BDS_G1_QUEUE="+queueName,
		"BDS_G1_MAIL_FILE="+mailFile,
	)
	if err := cmd.Start(); err != nil {
		logFile.Close()
		t.Fatal(err)
	}
	stopped := false
	stop := func() {
		if stopped {
			return
		}
		stopped = true
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		_ = logFile.Close()
		if t.Failed() {
			data, _ := os.ReadFile(logFile.Name())
			t.Logf("%s worker logs: %s", role, data)
		}
	}
	t.Cleanup(stop)
	return cmd, stop
}

func g1MailMessages(path string) ([]string, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	raw := strings.TrimSpace(string(b))
	if raw == "" {
		return nil, nil
	}
	return strings.Split(raw, "\n"), nil
}

// This process-level integration gate is deliberately opt-in. Asynq v0.23
// checks expired leases once a minute, with additional clock-skew protection.
// Do not add several minutes to every package test or every race pass.
func TestG1WorkerCrashAfterMailDeliveryAndAutomaticRecovery(t *testing.T) {
	if os.Getenv("BDS_G1_PROCESS_TEST") != "1" {
		t.Skip("set BDS_G1_PROCESS_TEST=1 in isolated integration CI")
	}
	if os.Getenv("BDS_TEST_MODE") != "disposable" || os.Getenv("BDS_TEST_REDIS_ADDR") == "" {
		t.Fatal("requires disposable PostgreSQL and Redis")
	}
	pool := isolatedOutboxPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 215*time.Second)
	defer cancel()
	publicConfig, err := pgxpool.ParseConfig(os.Getenv("BDS_TEST_DB_URL"))
	if err != nil || publicConfig.ConnConfig.Database != "bdspro_test" {
		t.Fatal("refusing non-test PostgreSQL")
	}
	publicPool, err := pgxpool.NewWithConfig(ctx, publicConfig)
	if err != nil {
		t.Fatal(err)
	}
	defer publicPool.Close()
	store := db.NewStore(publicPool)
	account, err := account_model.Register(ctx, store, account_model.RegisterParams{
		Email: uuid.NewString() + "@example.test", PasswordHash: "initial-test-hash",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.MarkEmailVerified(ctx, db.MarkEmailVerifiedParams{ID: account.ID, Email: account.Email}); err != nil {
		t.Fatal(err)
	}
	queueName := "g1_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	taskID := "g1-crash-" + uuid.NewString()
	mailFile := t.TempDir() + "/mail-sink.txt"
	var outboxID int64
	err = pool.QueryRow(ctx, `
		INSERT INTO job_outbox(task_type, payload, queue_name, max_retry, run_at, task_id)
		VALUES ('task:send_password_reset', jsonb_build_object('account_id', $1::bigint, 'reset_version', 0),
			$2, 3, now(), $3)
		RETURNING id`, account.ID, queueName, taskID).Scan(&outboxID)
	if err != nil {
		t.Fatal(err)
	}
	options := asynq.RedisClientOpt{Addr: os.Getenv("BDS_TEST_REDIS_ADDR"), DB: 15}
	enqueuer := queue.NewRedisEnqueuer(options)
	defer enqueuer.Close()
	n, err := NewOutboxDispatcher(db.NewStore(pool), enqueuer).Dispatch(ctx)
	if err != nil || n != 1 || !outboxQueued(t, ctx, pool, outboxID) {
		t.Fatalf("outbox was not committed: n=%d err=%v", n, err)
	}
	inspector := asynq.NewInspector(options)
	defer inspector.Close()
	defer func() {
		if err := inspector.DeleteTask(queueName, taskID); err != nil {
			t.Logf("cleanup isolated task: %v", err)
		}
	}()
	_, stopFirst := g1LaunchChild(t, "blocked", queueName, mailFile)
	// Ensure the first child performed a durable mail-sink side effect and
	// is still holding the active task before killing the process.
	for {
		messages, readErr := g1MailMessages(mailFile)
		if readErr != nil {
			t.Fatal(readErr)
		}
		if len(messages) == 1 {
			info, err := inspector.GetTaskInfo(queueName, taskID)
			if err != nil || info == nil || fmt.Sprint(info.State) != "active" {
				t.Fatalf("expected in-flight task, got info=%+v err=%v", info, err)
			}
			break
		}
		if len(messages) > 1 {
			t.Fatal("unexpected duplicate before first worker was killed")
		}
		select {
		case <-ctx.Done():
			t.Fatal("first worker never performed mail side effect")
		case <-time.After(100 * time.Millisecond):
		}
	}
	stopFirst() // SIGKILL: no graceful handler return or Asynq acknowledgement.
	// The task must remain durable in Redis despite the dead process.
	info, err := inspector.GetTaskInfo(queueName, taskID)
	if err != nil || info == nil {
		t.Fatalf("crash lost durable task: info=%+v err=%v", info, err)
	}
	_, stopSecond := g1LaunchChild(t, "recovered", queueName, mailFile)
	defer stopSecond()
	for {
		info, err := inspector.GetTaskInfo(queueName, taskID)
		if err == nil && info != nil && fmt.Sprint(info.State) == "completed" {
			break
		}
		select {
		case <-ctx.Done():
			t.Fatalf("lease was not recovered by replacement worker: state=%v err=%v", info, err)
		case <-time.After(250 * time.Millisecond):
		}
	}
	messages, err := g1MailMessages(mailFile)
	if err != nil || len(messages) != 2 {
		t.Fatalf("expected one mail side effect per worker execution: messages=%d err=%v", len(messages), err)
	}
	tokenPattern := regexp.MustCompile(`token=([A-Za-z0-9_-]{43})`)
	proofs := make([][]byte, 0, len(messages))
	for _, message := range messages {
		match := tokenPattern.FindStringSubmatch(message)
		if len(match) != 2 {
			t.Fatal("worker produced mail without a reset token")
		}
		digest := sha256.Sum256([]byte(match[1]))
		proofs = append(proofs, digest[:])
	}
	if string(proofs[0]) == string(proofs[1]) {
		t.Fatal("current password-reset contract unexpectedly reused one proof")
	}
	for i, proof := range proofs {
		var exists bool
		if err := publicPool.QueryRow(ctx,
			"SELECT EXISTS(SELECT 1 FROM password_reset_tokens WHERE account_id=$1 AND token_hash=$2)",
			account.ID, proof).Scan(&exists); err != nil || !exists {
			t.Fatalf("attempt %s missing durable reset proof: %v exists=%t", strconv.Itoa(i), err, exists)
		}
	}
	if err := auth_model.ResetPassword(ctx, store, proofs[0], "replacement-test-hash"); err != nil {
		t.Fatal(err)
	}
	for _, proof := range proofs {
		if err := auth_model.ResetPassword(ctx, store, proof, "forbidden-test-hash"); !errors.Is(err, auth_model.ErrPasswordResetInvalid) {
			t.Fatalf("redelivered reset proof remained valid after recovery: %v", err)
		}
	}
}
