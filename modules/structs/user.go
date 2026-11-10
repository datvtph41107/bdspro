package structs

import (
	"time"

	"github.com/google/uuid"
)

type UserResponse struct {
	AccountID       int64     `json:"account_id"`
	PersonID        int64     `json:"person_id"`
	Email           string    `json:"email"`
	CreatedAt       time.Time `json:"created_at"`
	IsEmailVerified bool      `json:"is_email_verified"`
}

type LoginResponse struct {
	SessionID             string       `json:"session_id"`
	AccessToken           string       `json:"access_token"`
	AccessTokenExpiresAt  time.Time    `json:"access_token_expires_at"`
	RefreshToken          string       `json:"refresh_token"`
	RefreshTokenExpiresAt time.Time    `json:"refresh_token_expires_at"`
	User                  UserResponse `json:"user"`
}

type RenewAccessTokenResponse struct {
	AccessToken          string    `json:"access_token"`
	AccessTokenExpiresAt time.Time `json:"access_token_expires_at"`
}

// ConnectionResponse is the public wire contract; it intentionally does not
// embed a generated SQLC row or expose account identifiers or refresh secrets.
type ConnectionResponse struct {
	ID        int64     `json:"id"`
	SessionID string    `json:"session_id"`
	Action    string    `json:"action"`
	IPAddress string    `json:"ip_address"`
	UserAgent string    `json:"user_agent"`
	CreatedAt time.Time `json:"created_at"`
}

// SessionResponse preserves the existing first-party /users/me/sessions JSON
// contract, without leaking an auth session's refresh-token fingerprint.
type SessionResponse struct {
	ID        uuid.UUID `json:"id"`
	UserAgent string    `json:"user_agent"`
	ClientIP  string    `json:"client_ip"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}
