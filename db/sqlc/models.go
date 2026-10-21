package db

import (
    "time"
    "github.com/google/uuid"
)

type Party struct { ID int64 `json:"id"` }
type Person struct { PartyID int64 `json:"party_id"` }

type Account struct {
    ID           int64     `json:"id"`
    PersonID     int64     `json:"person_id"`
    Email        string    `json:"email"`
    PasswordHash string    `json:"password_hash"`
    CreatedAt    time.Time `json:"created_at"`
}

type Session struct {
    ID           uuid.UUID `json:"id"`
    AccountID    int64     `json:"account_id"`
    RefreshToken string    `json:"refresh_token"`
    UserAgent    string    `json:"user_agent"`
    ClientIp     string    `json:"client_ip"`
    IsBlocked    bool      `json:"is_blocked"`
    CreatedAt    time.Time `json:"created_at"`
    ExpiresAt    time.Time `json:"expires_at"`
}
