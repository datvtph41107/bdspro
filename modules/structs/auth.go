package structs

type LoginRequest struct {
	Identifier   string `json:"identifier" binding:"required,max=254"`
	Password     string `json:"password" binding:"required,max=512"`
	StaySignedIn bool   `json:"stay_signed_in"`
}

type PasswordChangeRequest struct {
	CurrentPassword string `json:"current_password" binding:"required,max=512"`
	NewPassword     string `json:"new_password" binding:"required,max=512"`
}

type PasswordResetRequest struct {
	Email string `json:"email" binding:"required,max=254"`
}

type PasswordResetConfirm struct {
	ResetToken  string `json:"reset_token" binding:"required,max=128"`
	NewPassword string `json:"new_password" binding:"required,max=512"`
}

type RegistrationStart struct {
	Email string `json:"email" binding:"required,max=254"`
}

type RegistrationResend struct {
	ChallengeID string `json:"challenge_id" binding:"required,max=36"`
}

type RegistrationVerify struct {
	ChallengeID string `json:"challenge_id" binding:"required,max=36"`
	Code        string `json:"code" binding:"required,max=16"`
}

type RegistrationComplete struct {
	RegistrationToken string `json:"registration_token" binding:"required,max=128"`
	Username          string `json:"username" binding:"required,max=30"`
	Password          string `json:"password" binding:"required,max=512"`
	StaySignedIn      bool   `json:"stay_signed_in"`
	TermsVersion      string `json:"terms_version" binding:"required,max=64"`
	MarketingOptIn    bool   `json:"marketing_opt_in"`
}

type UpdateUsernameRequest struct {
	Username string `json:"username" binding:"required,max=30"`
}

type PhoneStart struct {
	Phone       string `json:"phone" binding:"required,max=32"`
	CountryCode string `json:"country_code" binding:"max=2"`
}

type PhoneVerify struct {
	ChallengeID  string `json:"challenge_id" binding:"required,max=36"`
	Code         string `json:"code" binding:"required,max=16"`
	StaySignedIn bool   `json:"stay_signed_in"`
}

type PhoneComplete struct {
	RegistrationToken string `json:"registration_token" binding:"required,max=128"`
	Username          string `json:"username" binding:"required,max=30"`
	StaySignedIn      bool   `json:"stay_signed_in"`
	TermsVersion      string `json:"terms_version" binding:"required,max=64"`
	MarketingOptIn    bool   `json:"marketing_opt_in"`
}

type GoogleComplete struct {
	RegistrationToken string `json:"registration_token" binding:"required,max=128"`
	Username          string `json:"username" binding:"required,max=30"`
	TermsVersion      string `json:"terms_version" binding:"required,max=64"`
	MarketingOptIn    bool   `json:"marketing_opt_in"`
}

type PasswordReauthRequest struct {
	Password string `json:"password" binding:"required,max=512"`
	Action   string `json:"action" binding:"required,max=64"`
}

type RecentAuthRequest struct {
	ReauthenticationProof string `json:"reauthentication_proof" binding:"required,max=128"`
}

type PasswordSetRequest struct {
	NewPassword           string `json:"new_password" binding:"required,max=512"`
	ReauthenticationProof string `json:"reauthentication_proof" binding:"required,max=128"`
}

type EmailLinkStart struct {
	Email                 string `json:"email" binding:"required,max=254"`
	ReauthenticationProof string `json:"reauthentication_proof" binding:"required,max=128"`
}

type PhoneLinkStart struct {
	Phone                 string `json:"phone" binding:"required,max=32"`
	CountryCode           string `json:"country_code" binding:"max=2"`
	ReauthenticationProof string `json:"reauthentication_proof" binding:"required,max=128"`
}

type ContactLinkVerify struct {
	ChallengeID string `json:"challenge_id" binding:"required,max=36"`
	Code        string `json:"code" binding:"required,max=16"`
}

type GoogleReauthRequest struct {
	Action string `json:"action" binding:"required,max=64"`
}
