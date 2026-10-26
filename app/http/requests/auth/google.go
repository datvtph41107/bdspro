package authrequest

type GoogleLinkBody struct {
    Password string `json:"password" binding:"required"`
}
