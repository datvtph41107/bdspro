package authrequest

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
