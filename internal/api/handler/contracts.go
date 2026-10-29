package handler

type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8,max=72"`
}
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}
type TokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}
type VerifyEmailQuery struct {
	EmailID    int64  `form:"email_id" binding:"required,min=1"`
	SecretCode string `form:"secret_code" binding:"required"`
}
type GoogleLinkRequest struct {
	Password string `json:"password" binding:"required"`
}
type PhoneBindRequest struct {
	Phone    string `json:"phone" binding:"required"`
	Password string `json:"password" binding:"required"`
}
type PhoneVerifyRequest struct {
	Phone string `json:"phone" binding:"required"`
	Code  string `json:"code" binding:"required"`
}
type CreateOrganizationRequest struct {
	Name string `json:"name" binding:"required"`
}
type UpdateOrganizationRequest struct {
	Name *string `json:"name"`
}
type AddOrganizationMemberRequest struct {
	PersonID int64 `json:"person_id" binding:"required,min=1"`
}

type PhoneStartRequest struct {
	Phone string `json:"phone" binding:"required"`
}

type PasswordResetRequest struct {
	Email string `json:"email" binding:"required,email"`
}
type PasswordResetConfirmRequest struct {
	Token    string `json:"token" form:"token" binding:"required"`
	Password string `json:"password" form:"password" binding:"required,min=8,max=72"`
}
