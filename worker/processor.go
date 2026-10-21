package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"log/slog"
	"net/url"

	db "github.com/datvtph41107/bdspro/db/sqlc"
	"github.com/datvtph41107/bdspro/mail"
	"github.com/datvtph41107/bdspro/util"
	"github.com/hibiken/asynq"
)

type RedisTaskProcessor struct {
	server         *asynq.Server
	store          db.Store
	mailer         mail.EmailSender
	verifyEmailURL string
}

func NewRedisTaskProcessor(options asynq.RedisClientOpt, store db.Store, mailer mail.EmailSender, verifyEmailURL string) *RedisTaskProcessor {
	server := asynq.NewServer(options, asynq.Config{
		Queues: map[string]int{QueueCritical: 10},
		ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
			slog.ErrorContext(ctx, "worker task failed", "task_type", task.Type(), "error", err)
		}),
	})
	return &RedisTaskProcessor{server: server, store: store, mailer: mailer, verifyEmailURL: verifyEmailURL}
}

func (p *RedisTaskProcessor) Start() error {
	mux := asynq.NewServeMux()
	mux.HandleFunc(TaskSendVerifyEmail, p.ProcessTaskSendVerifyEmail)
	return p.server.Start(mux)
}

func (p *RedisTaskProcessor) Shutdown() {
	p.server.Shutdown()
}

func (p *RedisTaskProcessor) ProcessTaskSendVerifyEmail(ctx context.Context, task *asynq.Task) error {
	var payload PayloadSendVerifyEmail
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("decode verification task: %w", errors.Join(err, asynq.SkipRetry))
	}
	account, err := p.store.GetAccountByID(ctx, payload.AccountID)
	if err != nil {
		return err
	}
	secret, err := util.RandomSecret()
	if err != nil {
		return err
	}
	record, err := p.store.CreateVerifyEmail(ctx, db.CreateVerifyEmailParams{
		AccountID: account.ID, Email: account.Email, SecretCode: secret,
	})
	if err != nil {
		return err
	}
	link, err := url.Parse(p.verifyEmailURL)
	if err != nil {
		return fmt.Errorf("invalid verification URL: %w", err)
	}
	query := link.Query()
	query.Set("email_id", fmt.Sprint(record.ID))
	query.Set("secret_code", secret)
	link.RawQuery = query.Encode()
	content := fmt.Sprintf("Welcome to BDSPro. Please verify your email: <a href=%q>Verify email</a>", html.EscapeString(link.String()))
	return p.mailer.SendEmail("Verify your BDSPro email", content, []string{account.Email}, nil, nil, nil)
}
