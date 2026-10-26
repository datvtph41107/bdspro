package authrequest

// Context keys bridge one validated request and its controller.
// The body/query structs remain ordinary typed Go data.
const (
    LoginKey = "request.login"
    RegisterKey = "request.register"
    TokenKey = "request.token"
    VerifyEmailKey = "request.verify_email"
)
