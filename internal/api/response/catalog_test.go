package response

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apperrors "github.com/datvtph41107/bdspro/internal/errors"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

func TestAllApplicationFailuresKeepTheirPublicContract(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		status  int
		code    string
		message string
	}{
		{name: "InvalidResetToken", err: apperrors.InvalidResetToken, status: http.StatusBadRequest, code: "auth.invalid_reset_token", message: "invalid or expired reset token"},
		{name: "InvalidPassword", err: apperrors.InvalidPassword, status: http.StatusBadRequest, code: "auth.invalid_password", message: "invalid request"},
		{name: "AccountUnavailable", err: apperrors.AccountUnavailable, status: http.StatusUnauthorized, code: "auth.account_unavailable", message: "account unavailable"},
		{name: "NameRequired", err: apperrors.NameRequired, status: http.StatusBadRequest, code: "http.name_required", message: "name required"},
		{name: "InvalidOrganization", err: apperrors.InvalidOrganization, status: http.StatusBadRequest, code: "http.invalid_organization", message: "invalid organization"},
		{name: "InvalidVerificationLink", err: apperrors.InvalidVerificationLink, status: http.StatusBadRequest, code: "auth.invalid_verification_link", message: "invalid or expired verification link"},
		{name: "InvalidRefreshToken", err: apperrors.InvalidRefreshToken, status: http.StatusUnauthorized, code: "auth.invalid_refresh_token", message: "invalid refresh token"},
		{name: "SessionNotFound", err: apperrors.SessionNotFound, status: http.StatusNotFound, code: "auth.session_not_found", message: "session not found"},
		{name: "SessionUnavailable", err: apperrors.SessionUnavailable, status: http.StatusUnauthorized, code: "auth.session_unavailable", message: "session unavailable"},
		{name: "InvalidCredentials", err: apperrors.InvalidCredentials, status: http.StatusUnauthorized, code: "auth.invalid_credentials", message: "invalid credentials"},
		{name: "InvalidEmail", err: apperrors.InvalidEmail, status: http.StatusBadRequest, code: "auth.invalid_email", message: "invalid email"},
		{name: "EmailAlreadyLinked", err: apperrors.EmailAlreadyLinked, status: http.StatusConflict, code: "auth.email_already_linked", message: "account already has an email"},
		{name: "InvalidEmailLink", err: apperrors.InvalidEmailLink, status: http.StatusBadRequest, code: "auth.invalid_email_link", message: "invalid or expired email link"},
		{name: "EmailLinkCooldown", err: apperrors.EmailLinkCooldown, status: http.StatusTooManyRequests, code: "auth.email_link_cooldown", message: "email verification temporarily unavailable"},
		{name: "EmailRegistered", err: apperrors.EmailRegistered, status: http.StatusConflict, code: "auth.email_registered", message: "email already registered"},
		{name: "OrganizationNotFound", err: apperrors.OrganizationNotFound, status: http.StatusNotFound, code: "organization.not_found", message: "organization not found"},
		{name: "OrganizationOwnerRequired", err: apperrors.OrganizationOwnerRequired, status: http.StatusForbidden, code: "organization.owner_required", message: "organization owner required"},
		{name: "MembershipExists", err: apperrors.MembershipExists, status: http.StatusConflict, code: "organization.membership_exists", message: "membership already exists"},
		{name: "PersonUnavailable", err: apperrors.PersonUnavailable, status: http.StatusNotFound, code: "organization.person_unavailable", message: "person unavailable"},
		{name: "GoogleIdentityAlreadyLinked", err: apperrors.GoogleIdentityAlreadyLinked, status: http.StatusConflict, code: "auth.google_identity_already_linked", message: "Google identity already linked"},
		{name: "GoogleAccountAlreadyLinked", err: apperrors.GoogleAccountAlreadyLinked, status: http.StatusConflict, code: "auth.google_account_already_linked", message: "account already linked"},
		{name: "GoogleIdentityUnlinked", err: apperrors.GoogleIdentityUnlinked, status: http.StatusUnauthorized, code: "auth.google_identity_unlinked", message: "Google identity not linked"},
		{name: "GoogleEmailUnverified", err: apperrors.GoogleEmailUnverified, status: http.StatusUnauthorized, code: "auth.google_email_unverified", message: "Google email is not verified"},
		{name: "GoogleEmailAlreadyRegistered", err: apperrors.GoogleEmailAlreadyRegistered, status: http.StatusConflict, code: "auth.google_email_registered", message: "email already registered; sign in and link Google"},
		{name: "InvalidPhone", err: apperrors.InvalidPhone, status: http.StatusBadRequest, code: "auth.invalid_phone", message: "invalid phone"},
		{name: "PhoneRegistered", err: apperrors.PhoneRegistered, status: http.StatusConflict, code: "auth.phone_already_registered", message: "phone number already registered"},
		{name: "PhoneUnavailable", err: apperrors.PhoneUnavailable, status: http.StatusConflict, code: "auth.phone_unavailable", message: "phone unavailable"},
		{name: "OTPInvalid", err: apperrors.OTPInvalid, status: http.StatusUnauthorized, code: "auth.otp_invalid", message: "invalid or expired code"},
		{name: "OTPCooldown", err: apperrors.OTPCooldown, status: http.StatusTooManyRequests, code: "auth.otp_cooldown", message: "code temporarily unavailable"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			failure, ok := classifyApplicationError(tt.err)
			if !ok {
				t.Fatal("application failure not classified")
			}
			if failure.Status != tt.status || failure.Code != tt.code || failure.Message != tt.message {
				t.Fatalf("got %+v, want status=%d code=%q message=%q", failure, tt.status, tt.code, tt.message)
			}
		})
	}
}

func TestWrappedApplicationFailureRetainsCauseAndHTTPContract(t *testing.T) {
	cause := errors.New("private db constraint detail")
	err := apperrors.Wrap(apperrors.MembershipExists, cause)
	if !errors.Is(err, apperrors.MembershipExists) || !errors.Is(err, cause) {
		t.Fatalf("both error identities must be preserved: %v", err)
	}
	failure, ok := classifyApplicationError(err)
	if !ok || failure.Status != http.StatusConflict || failure.Code != "organization.membership_exists" {
		t.Fatalf("incorrect public contract: %+v, ok=%t", failure, ok)
	}
	if _, ok := classifyApplicationError(errors.New("unknown database error")); ok {
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
