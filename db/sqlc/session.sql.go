package db

import (
    "context"
    "time"
    "github.com/google/uuid"
)

type CreateSessionParams struct {
    ID           uuid.UUID `json:"id"`
    AccountID    int64     `json:"account_id"`
    RefreshToken string    `json:"refresh_token"`
    UserAgent    string    `json:"user_agent"`
    ClientIp     string    `json:"client_ip"`
    IsBlocked    bool      `json:"is_blocked"`
    ExpiresAt    time.Time `json:"expires_at"`
}

const createSession = `INSERT INTO sessions
(id,account_id,refresh_token,user_agent,client_ip,is_blocked,expires_at)
VALUES ($1,$2,$3,$4,$5,$6,$7)
RETURNING id,account_id,refresh_token,user_agent,client_ip,is_blocked,created_at,expires_at`
func (q *Queries) CreateSession(ctx context.Context, arg CreateSessionParams) (Session, error) {
    var s Session
    err := q.db.QueryRow(ctx, createSession,
        arg.ID, arg.AccountID, arg.RefreshToken, arg.UserAgent,
        arg.ClientIp, arg.IsBlocked, arg.ExpiresAt,
    ).Scan(&s.ID, &s.AccountID, &s.RefreshToken, &s.UserAgent,
        &s.ClientIp, &s.IsBlocked, &s.CreatedAt, &s.ExpiresAt)
    return s, err
}

const getSession = `SELECT id,account_id,refresh_token,user_agent,client_ip,is_blocked,created_at,expires_at
FROM sessions WHERE id=$1 LIMIT 1`
func (q *Queries) GetSession(ctx context.Context, id uuid.UUID) (Session, error) {
    var s Session
    err := q.db.QueryRow(ctx, getSession, id).Scan(
        &s.ID, &s.AccountID, &s.RefreshToken, &s.UserAgent,
        &s.ClientIp, &s.IsBlocked, &s.CreatedAt, &s.ExpiresAt)
    return s, err
}
