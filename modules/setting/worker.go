package setting

import (
	"fmt"
	"strconv"
)

// WorkerSettings has exactly the runtime capabilities required to deliver
// mail, consume Redis/Asynq tasks and prune PostgreSQL audit histories.
// It intentionally excludes HTTP listener, migration, phone/OIDC client,
// CORS and JWT access/refresh TTL configuration.
type WorkerSettings struct {
	DatabaseCommandSettings
	Environment        string
	RedisAddress       string
	TokenSymmetricKey  string
	AuditRetentionDays int
	Queue              QueueSettings

	EmailSenderName     string
	EmailSenderAddress  string
	EmailSenderPassword string
	SMTPAddress         string
	SMTPAuth            bool
	VerifyEmailURL      string
}

func LoadWorkerSettings(filename string) (WorkerSettings, error) {
	provider, err := InitCfgProvider(filename)
	if err != nil {
		return WorkerSettings{}, err
	}
	database, err := databaseCommandSettings(provider)
	if err != nil {
		return WorkerSettings{}, err
	}
	cfg := WorkerSettings{
		DatabaseCommandSettings: database,
		Environment:             LoadCommonSettings(provider).RunMode,
		RedisAddress:            provider.String("redis", "ADDR"),
		TokenSymmetricKey:       provider.String("security", "TOKEN_SYMMETRIC_KEY"),
		EmailSenderName:         provider.String("mailer", "EMAIL_SENDER_NAME"),
		EmailSenderAddress:      provider.String("mailer", "EMAIL_SENDER_ADDRESS"),
		EmailSenderPassword:     provider.String("mailer", "EMAIL_SENDER_PASSWORD"),
		SMTPAddress:             provider.String("mailer", "SMTP_ADDRESS"),
		VerifyEmailURL:          provider.String("mailer", "VERIFY_EMAIL_URL"),
	}
	cfg.Queue, err = LoadQueueSettings(provider)
	if err != nil {
		return cfg, err
	}
	if cfg.RedisAddress == "" {
		return cfg, fmt.Errorf("redis.ADDR is required for worker")
	}
	if err := validateTokenSigningKey(cfg.Environment, cfg.TokenSymmetricKey); err != nil {
		return cfg, err
	}
	cfg.AuditRetentionDays, err = LoadAuditRetentionDays(provider)
	if err != nil {
		return cfg, err
	}
	cfg.SMTPAuth, err = strconv.ParseBool(provider.StringDefault("mailer", "SMTP_AUTH", "false"))
	if err != nil {
		return cfg, fmt.Errorf("SMTP_AUTH: %w", err)
	}
	return cfg, nil
}
