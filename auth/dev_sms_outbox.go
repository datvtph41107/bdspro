package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DevelopmentSMSOutbox implements the SAME SMSSender contract as the real
// provider. This intentionally stores cleartext OTP only in a disposable
// local development database. Never construct it outside development.
type DevelopmentSMSOutbox struct {
	pool *pgxpool.Pool
}

// NewDevelopmentSMSOutbox refuses to silently create schema at application
// startup. Provision db/dev_sms_outbox.sql explicitly against a local DB.
func NewDevelopmentSMSOutbox(ctx context.Context, pool *pgxpool.Pool) (*DevelopmentSMSOutbox, error) {
	if pool == nil {
		return nil, errors.New("dev SMS outbox requires database pool")
	}
	var relation *string
	if err := pool.QueryRow(ctx, "SELECT to_regclass('public.dev_sms_outbox')::text").Scan(&relation); err != nil {
		return nil, fmt.Errorf("inspect dev SMS outbox: %w", err)
	}
	if relation == nil {
		return nil, errors.New("dev_sms_outbox missing: apply db/dev_sms_outbox.sql to the local development database")
	}
	return &DevelopmentSMSOutbox{pool: pool}, nil
}

func (out *DevelopmentSMSOutbox) SendOTP(ctx context.Context, phone, code string) error {
	// This replaces transport ONLY; authentication state is stored separately
	// as a keyed HMAC in its domain challenge table.
	_, err := out.pool.Exec(ctx, `
		WITH cleanup AS (
			DELETE FROM dev_sms_outbox WHERE expires_at <= now()
		)
		INSERT INTO dev_sms_outbox (phone, otp_code) VALUES ($1, $2)
	`, phone, code)
	if err != nil {
		return fmt.Errorf("write development SMS outbox: %w", err)
	}
	return nil
}
