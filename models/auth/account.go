package auth

type CurrentUser struct {
	ID          string      `json:"id"`
	PersonID    string      `json:"person_id"`
	Username    *string     `json:"username"`
	DisplayName string      `json:"display_name"`
	AvatarURL   *string     `json:"avatar_url"`
	Status      string      `json:"status"`
	Email       *string     `json:"email"`
	Phone       *string     `json:"phone"`
	Credentials Credentials `json:"credentials"`
	Onboarding  Onboarding  `json:"onboarding"`
}

type Credentials struct {
	EmailVerified bool `json:"email_verified"`
	PhoneVerified bool `json:"phone_verified"`
	PasswordSet   bool `json:"password_set"`
	GoogleLinked  bool `json:"google_linked"`
}

type Onboarding struct {
	Completed       bool     `json:"completed"`
	RequiredActions []string `json:"required_actions"`
}
