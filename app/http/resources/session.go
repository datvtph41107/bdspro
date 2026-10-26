package resources

import (
    "time"
    "github.com/datvtph41107/bdspro/auth"
    db "github.com/datvtph41107/bdspro/db/sqlc"
)

type User struct {
    AccountID int64 `json:"account_id"`
    Email string `json:"email"`
    CreatedAt time.Time `json:"created_at"`
    IsEmailVerified bool `json:"is_email_verified"`
}
func NewUser(account db.Account) User {
    return User{AccountID:account.ID,Email:account.Email,CreatedAt:account.CreatedAt,IsEmailVerified:account.IsEmailVerified}
}
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
