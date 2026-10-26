package resources

import (
    "time"
    "github.com/datvtph41107/bdspro/auth"
)

type Login struct {
    SessionID string `json:"session_id"`
    AccessToken string `json:"access_token"`
    AccessTokenExpiresAt time.Time `json:"access_token_expires_at"`
    RefreshToken string `json:"refresh_token"`
    RefreshTokenExpiresAt time.Time `json:"refresh_token_expires_at"`
    User User `json:"user"`
}

func NewLogin(result auth.LoginResult) Login {
    return Login{
        SessionID:result.Session.ID.String(),
        AccessToken:result.AccessToken,AccessTokenExpiresAt:result.AccessExpiresAt,
        RefreshToken:result.RefreshToken,RefreshTokenExpiresAt:result.RefreshExpiresAt,
        User:NewUser(result.Account),
    }
}
