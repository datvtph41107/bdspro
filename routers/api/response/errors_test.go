package response

import (
	organization_model "github.com/datvtph41107/bdspro/models/organization"
	account_model "github.com/datvtph41107/bdspro/models/account"
	auth_model "github.com/datvtph41107/bdspro/models/auth"
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

func TestApplicationErrorResponses(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		status  int
		code    string
		message string
	}{
		{name: "InvalidResetToken", err: auth_model.ErrInvalidResetToken, status: http.StatusBadRequest, code: "auth.invalid_reset_token", message: "invalid or expired reset token"},
		{name: "InvalidPassword", err: auth_model.ErrInvalidPassword, status: http.StatusBadRequest, code: "auth.invalid_password", message: "invalid request"},
		{name: "AccountUnavailable", err: account_model.ErrAccountUnavailable, status: http.StatusUnauthorized, code: "auth.account_unavailable", message: "account unavailable"},
		{name: "NameRequired", err: organization_model.ErrNameRequired, status: http.StatusBadRequest, code: "http.name_required", message: "name required"},
		{name: "InvalidOrganization", err: organization_model.ErrInvalidOrganization, status: http.StatusBadRequest, code: "http.invalid_organization", message: "invalid organization"},
		{name: "InvalidVerificationLink", err: auth_model.ErrInvalidVerificationLink, status: http.StatusBadRequest, code: "auth.invalid_verification_link", message: "invalid or expired verification link"},
		{name: "InvalidRefreshToken", err: auth_model.ErrInvalidRefreshToken, status: http.StatusUnauthorized, code: "auth.invalid_refresh_token", message: "invalid refresh token"},
		{name: "SessionNotFound", err: auth_model.ErrSessionNotFound, status: http.StatusNotFound, code: "auth.session_not_found", message: "session not found"},
		{name: "SessionUnavailable", err: auth_model.ErrSessionUnavailable, status: http.StatusUnauthorized, code: "auth.session_unavailable", message: "session unavailable"},
		{name: "InvalidCredentials", err: auth_model.ErrInvalidCredentials, status: http.StatusUnauthorized, code: "auth.invalid_credentials", message: "invalid credentials"},
		{name: "InvalidEmail", err: auth_model.ErrInvalidEmail, status: http.StatusBadRequest, code: "auth.invalid_email", message: "invalid email"},
		{name: "EmailAlreadyLinked", err: auth_model.ErrEmailAlreadyLinked, status: http.StatusConflict, code: "auth.email_already_linked", message: "account already has an email"},
		{name: "InvalidEmailLink", err: auth_model.ErrInvalidEmailLink, status: http.StatusBadRequest, code: "auth.invalid_email_link", message: "invalid or expired email link"},
		{name: "EmailLinkCooldown", err: auth_model.ErrEmailLinkCooldown, status: http.StatusTooManyRequests, code: "auth.email_link_cooldown", message: "email verification temporarily unavailable"},
		{name: "EmailRegistered", err: account_model.ErrEmailRegistered, status: http.StatusConflict, code: "auth.email_registered", message: "email already registered"},
		{name: "OrganizationNotFound", err: organization_model.ErrOrganizationNotFound, status: http.StatusNotFound, code: "organization.not_found", message: "organization not found"},
		{name: "OrganizationOwnerRequired", err: organization_model.ErrOrganizationOwnerRequired, status: http.StatusForbidden, code: "organization.owner_required", message: "organization owner required"},
		{name: "MembershipExists", err: organization_model.ErrMembershipExists, status: http.StatusConflict, code: "organization.membership_exists", message: "membership already exists"},
		{name: "PersonUnavailable", err: organization_model.ErrPersonUnavailable, status: http.StatusNotFound, code: "organization.person_unavailable", message: "person unavailable"},
		{name: "GoogleIdentityAlreadyLinked", err: auth_model.ErrGoogleIdentityAlreadyLinked, status: http.StatusConflict, code: "auth.google_identity_already_linked", message: "Google identity already linked"},
		{name: "GoogleAccountAlreadyLinked", err: auth_model.ErrGoogleAccountAlreadyLinked, status: http.StatusConflict, code: "auth.google_account_already_linked", message: "account already linked"},
		{name: "GoogleIdentityUnlinked", err: auth_model.ErrGoogleIdentityUnlinked, status: http.StatusUnauthorized, code: "auth.google_identity_unlinked", message: "Google identity not linked"},
		{name: "GoogleEmailUnverified", err: auth_model.ErrGoogleEmailUnverified, status: http.StatusUnauthorized, code: "auth.google_email_unverified", message: "Google email is not verified"},
		{name: "GoogleEmailAlreadyRegistered", err: auth_model.ErrGoogleEmailAlreadyRegistered, status: http.StatusConflict, code: "auth.google_email_registered", message: "email already registered; sign in and link Google"},
		{name: "InvalidPhone", err: auth_model.ErrInvalidPhone, status: http.StatusBadRequest, code: "auth.invalid_phone", message: "invalid phone"},
		{name: "PhoneRegistered", err: auth_model.ErrPhoneRegistered, status: http.StatusConflict, code: "auth.phone_already_registered", message: "phone number already registered"},
		{name: "PhoneUnavailable", err: auth_model.ErrPhoneUnavailable, status: http.StatusConflict, code: "auth.phone_unavailable", message: "phone unavailable"},
		{name: "OTPInvalid", err: auth_model.ErrOTPInvalid, status: http.StatusUnauthorized, code: "auth.otp_invalid", message: "invalid or expired code"},
		{name: "OTPCooldown", err: auth_model.ErrOTPCooldown, status: http.StatusTooManyRequests, code: "auth.otp_cooldown", message: "code temporarily unavailable"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, ok := getErrorResponse(tt.err)
			if !ok {
				t.Fatal("application failure not classified")
			}
			if result.Status != tt.status || result.Code != tt.code || result.Message != tt.message {
				t.Fatalf("got %+v, want status=%d code=%q message=%q", result, tt.status, tt.code, tt.message)
			}
		})
	}
}

func TestWrappedApplicationErrorResponse(t *testing.T) {
	cause := errors.New("private db constraint detail")
	err := errors.Join(organization_model.ErrMembershipExists, cause)
	if !errors.Is(err, organization_model.ErrMembershipExists) || !errors.Is(err, cause) {
		t.Fatalf("both error identities must be preserved: %v", err)
	}
	result, ok := getErrorResponse(err)
	if !ok || result.Status != http.StatusConflict || result.Code != "organization.membership_exists" {
		t.Fatalf("incorrect public contract: %+v, ok=%t", result, ok)
	}
	if _, ok := getErrorResponse(errors.New("unknown database error")); ok {
		t.Fatal("unexpected errors must not be public")
	}
}

func TestUnexpectedErrorDoesNotLeakIntoClientOrLog(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var logs bytes.Buffer
	logger := zerolog.New(&logs)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx.Request = request.WithContext(logger.WithContext(request.Context()))

	const secret = "secret-credential-that-must-not-appear"
	Error(ctx, errors.New(secret))

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d", recorder.Code)
	}
	if strings.Contains(recorder.Body.String(), secret) || strings.Contains(logs.String(), secret) {
		t.Fatalf("unexpected raw error exposure: response=%q log=%q", recorder.Body.String(), logs.String())
	}
	if !strings.Contains(recorder.Body.String(), `"code":"internal_error"`) {
		t.Fatalf("unexpected response: %s", recorder.Body.String())
	}
	if !strings.Contains(logs.String(), `"error_type"`) {
		t.Fatalf("missing safe diagnostic category: %s", logs.String())
	}
}
