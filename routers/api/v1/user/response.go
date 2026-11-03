package user

import (
	"time"

	db "github.com/datvtph41107/bdspro/models/db"
	auth_service "github.com/datvtph41107/bdspro/services/auth"
)

type UserResponse struct {
	AccountID       int64     `json:"account_id"`
	Email           string    `json:"email"`
	CreatedAt       time.Time `json:"created_at"`
	IsEmailVerified bool      `json:"is_email_verified"`
}

func newUserResponse(a db.Account) UserResponse {
	return UserResponse{
		AccountID:       a.ID,
		Email:           a.Email,
		CreatedAt:       a.CreatedAt,
		IsEmailVerified: a.IsEmailVerified,
	}
}

type LoginResponse struct {
	SessionID             string       `json:"session_id"`
	AccessToken           string       `json:"access_token"`
	AccessTokenExpiresAt  time.Time    `json:"access_token_expires_at"`
	RefreshToken          string       `json:"refresh_token"`
	RefreshTokenExpiresAt time.Time    `json:"refresh_token_expires_at"`
	User                  UserResponse `json:"user"`
}

func newLoginResponse(v auth_service.LoginResult) LoginResponse {
	return LoginResponse{
		SessionID:             v.Session.ID.String(),
		AccessToken:           v.AccessToken,
		AccessTokenExpiresAt:  v.AccessExpiresAt,
		RefreshToken:          v.RefreshToken,
		RefreshTokenExpiresAt: v.RefreshExpiresAt,
		User:                  newUserResponse(v.Account),
	}
}

type RenewAccessTokenResponse struct {
	AccessToken          string    `json:"access_token"`
	AccessTokenExpiresAt time.Time `json:"access_token_expires_at"`
}

