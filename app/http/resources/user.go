package resources

import (
	db "github.com/datvtph41107/bdspro/db/sqlc"
	"time"
)

type User struct {
	AccountID       int64     `json:"account_id"`
	Email           string    `json:"email"`
	CreatedAt       time.Time `json:"created_at"`
	IsEmailVerified bool      `json:"is_email_verified"`
}

func NewUser(account db.Account) User {
	return User{AccountID: account.ID, Email: account.Email, CreatedAt: account.CreatedAt, IsEmailVerified: account.IsEmailVerified}
}
