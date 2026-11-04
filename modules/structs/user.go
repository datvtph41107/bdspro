package structs

import "time"

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
