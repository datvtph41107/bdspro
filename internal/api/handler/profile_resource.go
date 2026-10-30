package handler

import (
	"time"

	db "github.com/datvtph41107/bdspro/internal/database"
)

type PersonProfileResponse struct {
	DisplayName string  `json:"display_name"`
	FirstName   string  `json:"first_name"`
	LastName    string  `json:"last_name"`
}

type AccountSecurityResponse struct {
	HasPassword bool `json:"has_password"`
	HasPhone    bool `json:"has_phone"`
	HasGoogle   bool `json:"has_google"`
}

type AccountSettingsResponse struct {
	Locale   string `json:"locale"`
	Timezone string `json:"timezone"`
	Currency string `json:"currency"`
}

type AccountProfileResponse struct {
	AccountID       int64                   `json:"account_id"`
	PersonID        int64                   `json:"person_id"`
	Email           string                  `json:"email"`
	IsEmailVerified bool                    `json:"is_email_verified"`
	CreatedAt       time.Time               `json:"created_at"`
	Profile         PersonProfileResponse   `json:"profile"`
	Security        AccountSecurityResponse `json:"security"`
	Settings        AccountSettingsResponse `json:"settings"`
}

func newAccountSettingsResponse(p db.GetAccountProfileRow) AccountSettingsResponse {
	return AccountSettingsResponse{
		Locale: p.Locale, Timezone: p.Timezone, Currency: p.Currency,
	}
}

func newAccountProfileResponse(p db.GetAccountProfileRow) AccountProfileResponse {
	return AccountProfileResponse{
		AccountID:       p.AccountID,
		PersonID:        p.PersonID,
		Email:           p.Email,
		IsEmailVerified: p.IsEmailVerified,
		CreatedAt:       p.CreatedAt,
		Profile: PersonProfileResponse{
			DisplayName: p.DisplayName, FirstName: p.FirstName,
			LastName: p.LastName,
		},
		Security: AccountSecurityResponse{
			HasPassword: p.HasPassword, HasPhone: p.HasPhone, HasGoogle: p.HasGoogle,
		},
		Settings: newAccountSettingsResponse(p),
	}
}
