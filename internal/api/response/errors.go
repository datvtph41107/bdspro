package response

import (
	"errors"
	"net/http"

	apperrors "github.com/datvtph41107/bdspro/internal/errors"
)

// errorResponse contains the HTTP status and safe client message for a Go error.
type errorResponse struct {
	Status  int
	Code    string
	Message string
}

// getErrorResponse maps application errors to the existing HTTP contract.
func getErrorResponse(err error) (errorResponse, bool) {
	var app *apperrors.Error
	if !errors.As(err, &app) {
		return errorResponse{}, false
	}
	switch app {
	case apperrors.InvalidResetToken:
		return errorResponse{Status: http.StatusBadRequest, Code: "auth.invalid_reset_token", Message: "invalid or expired reset token"}, true
	case apperrors.InvalidPassword:
		return errorResponse{Status: http.StatusBadRequest, Code: "auth.invalid_password", Message: "invalid request"}, true
	case apperrors.AccountUnavailable:
		return errorResponse{Status: http.StatusUnauthorized, Code: "auth.account_unavailable", Message: "account unavailable"}, true
	case apperrors.NameRequired:
		return errorResponse{Status: http.StatusBadRequest, Code: "http.name_required", Message: "name required"}, true
	case apperrors.InvalidOrganization:
		return errorResponse{Status: http.StatusBadRequest, Code: "http.invalid_organization", Message: "invalid organization"}, true
	case apperrors.InvalidVerificationLink:
		return errorResponse{Status: http.StatusBadRequest, Code: "auth.invalid_verification_link", Message: "invalid or expired verification link"}, true
	case apperrors.InvalidRefreshToken:
		return errorResponse{Status: http.StatusUnauthorized, Code: "auth.invalid_refresh_token", Message: "invalid refresh token"}, true
	case apperrors.SessionNotFound:
		return errorResponse{Status: http.StatusNotFound, Code: "auth.session_not_found", Message: "session not found"}, true
	case apperrors.SessionUnavailable:
		return errorResponse{Status: http.StatusUnauthorized, Code: "auth.session_unavailable", Message: "session unavailable"}, true
	case apperrors.InvalidCredentials:
		return errorResponse{Status: http.StatusUnauthorized, Code: "auth.invalid_credentials", Message: "invalid credentials"}, true
	case apperrors.InvalidEmail:
		return errorResponse{Status: http.StatusBadRequest, Code: "auth.invalid_email", Message: "invalid email"}, true
	case apperrors.EmailAlreadyLinked:
		return errorResponse{Status: http.StatusConflict, Code: "auth.email_already_linked", Message: "account already has an email"}, true
	case apperrors.InvalidEmailLink:
		return errorResponse{Status: http.StatusBadRequest, Code: "auth.invalid_email_link", Message: "invalid or expired email link"}, true
	case apperrors.EmailLinkCooldown:
		return errorResponse{Status: http.StatusTooManyRequests, Code: "auth.email_link_cooldown", Message: "email verification temporarily unavailable"}, true
	case apperrors.EmailRegistered:
		return errorResponse{Status: http.StatusConflict, Code: "auth.email_registered", Message: "email already registered"}, true
	case apperrors.OrganizationNotFound:
		return errorResponse{Status: http.StatusNotFound, Code: "organization.not_found", Message: "organization not found"}, true
	case apperrors.OrganizationOwnerRequired:
		return errorResponse{Status: http.StatusForbidden, Code: "organization.owner_required", Message: "organization owner required"}, true
	case apperrors.MembershipExists:
		return errorResponse{Status: http.StatusConflict, Code: "organization.membership_exists", Message: "membership already exists"}, true
	case apperrors.PersonUnavailable:
		return errorResponse{Status: http.StatusNotFound, Code: "organization.person_unavailable", Message: "person unavailable"}, true
	case apperrors.GoogleIdentityAlreadyLinked:
		return errorResponse{Status: http.StatusConflict, Code: "auth.google_identity_already_linked", Message: "Google identity already linked"}, true
	case apperrors.GoogleAccountAlreadyLinked:
		return errorResponse{Status: http.StatusConflict, Code: "auth.google_account_already_linked", Message: "account already linked"}, true
	case apperrors.GoogleIdentityUnlinked:
		return errorResponse{Status: http.StatusUnauthorized, Code: "auth.google_identity_unlinked", Message: "Google identity not linked"}, true
	case apperrors.GoogleEmailUnverified:
		return errorResponse{Status: http.StatusUnauthorized, Code: "auth.google_email_unverified", Message: "Google email is not verified"}, true
	case apperrors.GoogleEmailAlreadyRegistered:
		return errorResponse{Status: http.StatusConflict, Code: "auth.google_email_registered", Message: "email already registered; sign in and link Google"}, true
	case apperrors.InvalidPhone:
		return errorResponse{Status: http.StatusBadRequest, Code: "auth.invalid_phone", Message: "invalid phone"}, true
	case apperrors.PhoneRegistered:
		return errorResponse{Status: http.StatusConflict, Code: "auth.phone_already_registered", Message: "phone number already registered"}, true
	case apperrors.PhoneUnavailable:
		return errorResponse{Status: http.StatusConflict, Code: "auth.phone_unavailable", Message: "phone unavailable"}, true
	case apperrors.OTPInvalid:
		return errorResponse{Status: http.StatusUnauthorized, Code: "auth.otp_invalid", Message: "invalid or expired code"}, true
	case apperrors.OTPCooldown:
		return errorResponse{Status: http.StatusTooManyRequests, Code: "auth.otp_cooldown", Message: "code temporarily unavailable"}, true
	default:
		return errorResponse{}, false
	}
}
