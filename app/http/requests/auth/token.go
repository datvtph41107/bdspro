package authrequest

type TokenBody struct {
    RefreshToken string `json:"refresh_token" binding:"required"`
}
