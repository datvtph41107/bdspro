package structs

import "time"

type ProfileUpdateRequest struct {
	DisplayName *string `json:"display_name"`
	FirstName   *string `json:"first_name"`
	LastName    *string `json:"last_name"`
}

type SettingsUpdateRequest struct {
	Locale   *string `json:"locale"`
	Timezone *string `json:"timezone"`
	Currency *string `json:"currency"`
}

type PersonProfileResponse struct {
	DisplayName string `json:"display_name"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
}

type AccountSecurityResponse struct {
	HasPassword bool `json:"has_password"`
	HasPhone    bool `json:"has_phone"`
	HasOAuth2   bool `json:"has_oauth2"`
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
