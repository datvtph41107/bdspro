package api

import (
    "net/http"

    "github.com/gin-gonic/gin"
)

const (
    loginRequestContextKey = "request.login"
    registerRequestContextKey = "request.register"
    tokenRequestContextKey = "request.token"
    verifyEmailRequestContextKey = "request.verify_email"
)

// PreLogin validates credentials syntax at the HTTP boundary. Authentication
// itself stays in auth.Login and MUST happen before issuing a session.
func PreLogin() gin.HandlerFunc {
    return func(ctx *gin.Context) {
        var input loginUserRequest
        if err := ctx.ShouldBindJSON(&input); err != nil {
            invalidJSON(ctx, err)
            ctx.Abort()
            return
        }
        ctx.Set(loginRequestContextKey, input)
        ctx.Next()
    }
}

func PreRegister() gin.HandlerFunc {
    return func(ctx *gin.Context) {
        var input createUserRequest
        if err := ctx.ShouldBindJSON(&input); err != nil {
            invalidJSON(ctx, err)
            ctx.Abort()
            return
        }
        ctx.Set(registerRequestContextKey, input)
        ctx.Next()
    }
}

func PreToken() gin.HandlerFunc {
    return func(ctx *gin.Context) {
        var input refreshTokenRequest
        if err := ctx.ShouldBindJSON(&input); err != nil {
            invalidJSON(ctx, err)
            ctx.Abort()
            return
        }
        ctx.Set(tokenRequestContextKey, input)
        ctx.Next()
    }
}

func PreVerifyEmail() gin.HandlerFunc {
    return func(ctx *gin.Context) {
        var input verifyEmailRequest
        if err := ctx.ShouldBindQuery(&input); err != nil {
            ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid verification link"})
            return
        }
        ctx.Set(verifyEmailRequestContextKey, input)
        ctx.Next()
    }
}

// The typed values below are set only after prevalidation. The handlers cannot
// be reached without their corresponding Pre* route middleware.
func loginInput(ctx *gin.Context) loginUserRequest {
    return ctx.MustGet(loginRequestContextKey).(loginUserRequest)
}
func registerInput(ctx *gin.Context) createUserRequest {
    return ctx.MustGet(registerRequestContextKey).(createUserRequest)
}
func tokenInput(ctx *gin.Context) refreshTokenRequest {
    return ctx.MustGet(tokenRequestContextKey).(refreshTokenRequest)
}
func verifyEmailInput(ctx *gin.Context) verifyEmailRequest {
    return ctx.MustGet(verifyEmailRequestContextKey).(verifyEmailRequest)
}
