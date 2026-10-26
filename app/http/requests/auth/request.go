package authrequest

type RegisterBody struct {
    Email string `json:"email" binding:"required,email"`
    Password string `json:"password" binding:"required,min=8,max=72"`
}
type LoginBody struct {
    Email string `json:"email" binding:"required,email"`
    Password string `json:"password" binding:"required"`
}
type TokenBody struct {
    RefreshToken string `json:"refresh_token" binding:"required"`
}
type VerifyEmailQuery struct {
    EmailID int64 `form:"email_id" binding:"required,min=1"`
    SecretCode string `form:"secret_code" binding:"required"`
}
type PhoneBindBody struct {
    Phone string `json:"phone" binding:"required"`
    Password string `json:"password" binding:"required"`
}
type PhoneVerifyBody struct {
    Phone string `json:"phone" binding:"required"`
    Code string `json:"code" binding:"required"`
}
type PhoneStartBody struct {
    Phone string `json:"phone" binding:"required"`
}
type GoogleLinkBody struct {
    RedirectURI string `json:"redirect_uri"`
}
