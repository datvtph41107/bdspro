package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// HTTPFailure names a failure originating at the HTTP boundary.
// Its status and client-facing text belong only to the response package.
type HTTPFailure uint8

const (
	InvalidRequest HTTPFailure = iota
	RequestBodyTooLarge
	InternalServerError
	BearerRequired
	InvalidAccessToken
	AuthenticationRequired
	EmailVerificationRequired
	ContactVerificationRequired
	IdentityStoreUnavailable
	RateLimitUnavailable
	RateLimited
	InvalidVerificationLinkPage
	InvalidResetTokenPage
	InvalidSessionID
	InvalidLinkProof
	PhoneReauthenticationUnavailable
	GoogleUnavailable
	GoogleInvalidState
	GoogleStateExpired
	GoogleAuthorizationDeclined
	GoogleAuthorizationFailed
	GoogleInvalidIdentity
)

// Fail writes a canonical HTTP failure; callers do not choose its status or text.
func Fail(c *gin.Context, failure HTTPFailure) {
	var result Failure
	switch failure {
	case InvalidRequest:
		result = Failure{Status: http.StatusBadRequest, Code: "invalid_request", Message: "invalid request"}
	case RequestBodyTooLarge:
		result = Failure{Status: http.StatusRequestEntityTooLarge, Code: "request_body_too_large", Message: "request body too large"}
	case InternalServerError:
		result = Failure{Status: http.StatusInternalServerError, Code: "internal_error", Message: "internal server error"}
	case BearerRequired:
		result = Failure{Status: http.StatusUnauthorized, Code: "unauthenticated", Message: "missing or invalid bearer token"}
	case InvalidAccessToken:
		result = Failure{Status: http.StatusUnauthorized, Code: "invalid_access_token", Message: "invalid access token"}
	case AuthenticationRequired:
		result = Failure{Status: http.StatusUnauthorized, Code: "unauthenticated", Message: "authentication required"}
	case EmailVerificationRequired:
		result = Failure{Status: http.StatusForbidden, Code: "email_verification_required", Message: "email verification required"}
	case ContactVerificationRequired:
		result = Failure{Status: http.StatusForbidden, Code: "contact_verification_required", Message: "verified contact required"}
	case IdentityStoreUnavailable:
		result = Failure{Status: http.StatusServiceUnavailable, Code: "identity_store_unavailable", Message: "identity store unavailable"}
	case RateLimitUnavailable:
		result = Failure{Status: http.StatusServiceUnavailable, Code: "rate_limit_unavailable", Message: "authentication temporarily unavailable"}
	case RateLimited:
		result = Failure{Status: http.StatusTooManyRequests, Code: "rate_limited", Message: "too many requests"}
	case InvalidVerificationLinkPage:
		result = Failure{Status: http.StatusBadRequest, Code: "invalid_verification_link", Message: "invalid verification link"}
	case InvalidResetTokenPage:
		result = Failure{Status: http.StatusBadRequest, Code: "invalid_reset_token", Message: "invalid or expired reset token"}
	case InvalidSessionID:
		result = Failure{Status: http.StatusBadRequest, Code: "invalid_session_id", Message: "invalid session id"}
	case InvalidLinkProof:
		result = Failure{Status: http.StatusBadRequest, Code: "invalid_link_proof", Message: "provide exactly one reauthentication method"}
	case PhoneReauthenticationUnavailable:
		result = Failure{Status: http.StatusForbidden, Code: "phone_reauthentication_unavailable", Message: "phone reauthentication unavailable"}
	case GoogleUnavailable:
		result = Failure{Status: http.StatusServiceUnavailable, Code: "google_unavailable", Message: "Google sign-in unavailable"}
	case GoogleInvalidState:
		result = Failure{Status: http.StatusBadRequest, Code: "google_invalid_state", Message: "invalid Google authorization state"}
	case GoogleStateExpired:
		result = Failure{Status: http.StatusBadRequest, Code: "google_state_expired", Message: "expired Google authorization state"}
	case GoogleAuthorizationDeclined:
		result = Failure{Status: http.StatusUnauthorized, Code: "google_authorization_declined", Message: "Google authorization declined"}
	case GoogleAuthorizationFailed:
		result = Failure{Status: http.StatusUnauthorized, Code: "google_authorization_failed", Message: "Google authorization failed"}
	case GoogleInvalidIdentity:
		result = Failure{Status: http.StatusUnauthorized, Code: "google_invalid_identity", Message: "invalid Google identity"}
	default:
		result = Failure{Status: http.StatusInternalServerError, Code: "internal_error", Message: "internal server error"}
	}

	switch failure {
	case BearerRequired, AuthenticationRequired:
		c.Header("WWW-Authenticate", "Bearer")
	case InvalidAccessToken:
		c.Header("WWW-Authenticate", `Bearer error="invalid_token"`)
	}

	reject(c, result.Status, result.Code, result.Message)
}
