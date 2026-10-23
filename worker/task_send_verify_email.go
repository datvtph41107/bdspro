package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"net/url"

	db "github.com/datvtph41107/bdspro/db/sqlc"
	"github.com/datvtph41107/bdspro/util"
	"github.com/hibiken/asynq"
	"github.com/rs/zerolog/log"
)

const TaskSendVerifyEmail = "task:send_verify_email"

type PayloadSendVerifyEmail struct {
	AccountID int64 `json:"account_id"`
}

func (d *RedisTaskDistributor) DistributeTaskSendVerifyEmail(ctx context.Context, payload *PayloadSendVerifyEmail, opts ...asynq.Option) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal verification task: %w", err)
	}
	task := asynq.NewTask(TaskSendVerifyEmail, body, opts...)
	info, err := d.client.EnqueueContext(ctx, task)
	if err != nil {
		return err
	}
	log.Info().Str("type", task.Type()).Str("queue", info.Queue).Msg("task enqueued")
	return nil
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
	if err := p.mailer.SendEmail("Verify your BDSPro email", content, []string{account.Email}, nil, nil, nil); err != nil {
		return err
	}
	log.Info().Str("type", task.Type()).Msg("verification_email.completed")
	return nil
}
