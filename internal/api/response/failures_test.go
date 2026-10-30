package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// Each entry is a pre-existing public client contract, not the implementation source.
// Codes used by page handlers are intentionally not silently renamed to
// their similar-looking auth.* application error counterparts.
func TestHTTPFailureContracts(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name      string
		failure   HTTPFailure
		status    int
		code      string
		message   string
		challenge string
	}{
		{name: "InvalidRequest", failure: InvalidRequest, status: 400, code: "invalid_request", message: "invalid request", challenge: ""},
		{name: "RequestBodyTooLarge", failure: RequestBodyTooLarge, status: 413, code: "request_body_too_large", message: "request body too large", challenge: ""},
		{name: "InternalServerError", failure: InternalServerError, status: 500, code: "internal_error", message: "internal server error", challenge: ""},
		{name: "BearerRequired", failure: BearerRequired, status: 401, code: "unauthenticated", message: "missing or invalid bearer token", challenge: "Bearer"},
		{name: "InvalidAccessToken", failure: InvalidAccessToken, status: 401, code: "invalid_access_token", message: "invalid access token", challenge: "Bearer error=\"invalid_token\""},
		{name: "AuthenticationRequired", failure: AuthenticationRequired, status: 401, code: "unauthenticated", message: "authentication required", challenge: "Bearer"},
		{name: "EmailVerificationRequired", failure: EmailVerificationRequired, status: 403, code: "email_verification_required", message: "email verification required", challenge: ""},
		{name: "ContactVerificationRequired", failure: ContactVerificationRequired, status: 403, code: "contact_verification_required", message: "verified contact required", challenge: ""},
		{name: "IdentityStoreUnavailable", failure: IdentityStoreUnavailable, status: 503, code: "identity_store_unavailable", message: "identity store unavailable", challenge: ""},
		{name: "RateLimitUnavailable", failure: RateLimitUnavailable, status: 503, code: "rate_limit_unavailable", message: "authentication temporarily unavailable", challenge: ""},
		{name: "RateLimited", failure: RateLimited, status: 429, code: "rate_limited", message: "too many requests", challenge: ""},
		{name: "InvalidVerificationLinkPage", failure: InvalidVerificationLinkPage, status: 400, code: "invalid_verification_link", message: "invalid verification link", challenge: ""},
		{name: "InvalidResetTokenPage", failure: InvalidResetTokenPage, status: 400, code: "invalid_reset_token", message: "invalid or expired reset token", challenge: ""},
		{name: "InvalidSessionID", failure: InvalidSessionID, status: 400, code: "invalid_session_id", message: "invalid session id", challenge: ""},
		{name: "InvalidLinkProof", failure: InvalidLinkProof, status: 400, code: "invalid_link_proof", message: "provide exactly one reauthentication method", challenge: ""},
		{name: "PhoneReauthenticationUnavailable", failure: PhoneReauthenticationUnavailable, status: 403, code: "phone_reauthentication_unavailable", message: "phone reauthentication unavailable", challenge: ""},
		{name: "GoogleUnavailable", failure: GoogleUnavailable, status: 503, code: "google_unavailable", message: "Google sign-in unavailable", challenge: ""},
		{name: "GoogleInvalidState", failure: GoogleInvalidState, status: 400, code: "google_invalid_state", message: "invalid Google authorization state", challenge: ""},
		{name: "GoogleStateExpired", failure: GoogleStateExpired, status: 400, code: "google_state_expired", message: "expired Google authorization state", challenge: ""},
		{name: "GoogleAuthorizationDeclined", failure: GoogleAuthorizationDeclined, status: 401, code: "google_authorization_declined", message: "Google authorization declined", challenge: ""},
		{name: "GoogleAuthorizationFailed", failure: GoogleAuthorizationFailed, status: 401, code: "google_authorization_failed", message: "Google authorization failed", challenge: ""},
		{name: "GoogleInvalidIdentity", failure: GoogleInvalidIdentity, status: 401, code: "google_invalid_identity", message: "invalid Google identity", challenge: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequest(http.MethodGet, "/", nil)
			Fail(ctx, tt.failure)

			if recorder.Code != tt.status {
				t.Fatalf("status = %d, want %d", recorder.Code, tt.status)
			}
			var got struct {
				Code  string `json:"code"`
				Error string `json:"error"`
			}
			if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
				t.Fatal(err)
			}
			if got.Code != tt.code || got.Error != tt.message {
				t.Fatalf("public contract = %+v, want code=%q message=%q", got, tt.code, tt.message)
			}
			if recorder.Header().Get("WWW-Authenticate") != tt.challenge {
				t.Fatalf("WWW-Authenticate = %q, want %q", recorder.Header().Get("WWW-Authenticate"), tt.challenge)
			}
			if ctx.GetString(ErrorCodeKey) != tt.code {
				t.Fatalf("request error code = %q, want %q", ctx.GetString(ErrorCodeKey), tt.code)
			}
		})
	}
}
