package auth

import (
	"time"

	"github.com/google/uuid"
)

type Profile struct {
	DisplayName string `json:"display_name"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
}

type AccountSettings struct {
	Locale   string `json:"locale"`
	Timezone string `json:"timezone"`
	Currency string `json:"currency"`
}

type CredentialSummary struct {
	Email       *string     `json:"email"`
	Phone       *string     `json:"phone"`
	Credentials Credentials `json:"credentials"`
}

type Connection struct {
	ID        string    `json:"id"`
	SessionID uuid.UUID `json:"session_id"`
	Action    string    `json:"action"`
	IPAddress string    `json:"ip_address"`
	UserAgent string    `json:"user_agent"`
	CreatedAt time.Time `json:"created_at"`
}
