package response

import (
	"errors"
	"net/http"

	apperrors "github.com/datvtph41107/bdspro/internal/errors"
)

// Failure is the public HTTP representation of an application failure.
// The application error itself carries neither HTTP status nor client text.
type Failure struct {
	Status  int
	Code    string
	Message string
}

// classifyApplicationError is the sole HTTP mapping of application failures.
// Existing public codes and messages are preserved for client compatibility.
func classifyApplicationError(err error) (Failure, bool) {
	var app *apperrors.Error
	if !errors.As(err, &app) {
		return Failure{}, false
	}
	switch app {
	case apperrors.InvalidResetToken:
		return Failure{Status: http.StatusBadRequest, Code: "auth.invalid_reset_token", Message: "invalid or expired reset token"}, true
	case apperrors.InvalidPassword:
		return Failure{Status: http.StatusBadRequest, Code: "auth.invalid_password", Message: "invalid request"}, true
	case apperrors.AccountUnavailable:
		return Failure{Status: http.StatusUnauthorized, Code: "auth.account_unavailable", Message: "account unavailable"}, true
	case apperrors.NameRequired:
		return Failure{Status: http.StatusBadRequest, Code: "http.name_required", Message: "name required"}, true
	case apperrors.InvalidOrganization:
		return Failure{Status: http.StatusBadRequest, Code: "http.invalid_organization", Message: "invalid organization"}, true
	case apperrors.InvalidVerificationLink:
		return Failure{Status: http.StatusBadRequest, Code: "auth.invalid_verification_link", Message: "invalid or expired verification link"}, true
	case apperrors.InvalidRefreshToken:
		return Failure{Status: http.StatusUnauthorized, Code: "auth.invalid_refresh_token", Message: "invalid refresh token"}, true
	case apperrors.SessionNotFound:
		return Failure{Status: http.StatusNotFound, Code: "auth.session_not_found", Message: "session not found"}, true
	case apperrors.SessionUnavailable:
		return Failure{Status: http.StatusUnauthorized, Code: "auth.session_unavailable", Message: "session unavailable"}, true
	case apperrors.InvalidCredentials:
		return Failure{Status: http.StatusUnauthorized, Code: "auth.invalid_credentials", Message: "invalid credentials"}, true
	case apperrors.InvalidEmail:
		return Failure{Status: http.StatusBadRequest, Code: "auth.invalid_email", Message: "invalid email"}, true
	case apperrors.EmailAlreadyLinked:
		return Failure{Status: http.StatusConflict, Code: "auth.email_already_linked", Message: "account already has an email"}, true
	case apperrors.InvalidEmailLink:
		return Failure{Status: http.StatusBadRequest, Code: "auth.invalid_email_link", Message: "invalid or expired email link"}, true
	case apperrors.EmailLinkCooldown:
		return Failure{Status: http.StatusTooManyRequests, Code: "auth.email_link_cooldown", Message: "email verification temporarily unavailable"}, true
	case apperrors.EmailRegistered:
		return Failure{Status: http.StatusConflict, Code: "auth.email_registered", Message: "email already registered"}, true
	case apperrors.OrganizationNotFound:
		return Failure{Status: http.StatusNotFound, Code: "organization.not_found", Message: "organization not found"}, true
	case apperrors.OrganizationOwnerRequired:
		return Failure{Status: http.StatusForbidden, Code: "organization.owner_required", Message: "organization owner required"}, true
	case apperrors.MembershipExists:
		return Failure{Status: http.StatusConflict, Code: "organization.membership_exists", Message: "membership already exists"}, true
	case apperrors.PersonUnavailable:
		return Failure{Status: http.StatusNotFound, Code: "organization.person_unavailable", Message: "person unavailable"}, true
	case apperrors.GoogleIdentityAlreadyLinked:
		return Failure{Status: http.StatusConflict, Code: "auth.google_identity_already_linked", Message: "Google identity already linked"}, true
	case apperrors.GoogleAccountAlreadyLinked:
		return Failure{Status: http.StatusConflict, Code: "auth.google_account_already_linked", Message: "account already linked"}, true
	case apperrors.GoogleIdentityUnlinked:
		return Failure{Status: http.StatusUnauthorized, Code: "auth.google_identity_unlinked", Message: "Google identity not linked"}, true
	case apperrors.GoogleEmailUnverified:
		return Failure{Status: http.StatusUnauthorized, Code: "auth.google_email_unverified", Message: "Google email is not verified"}, true
	case apperrors.GoogleEmailAlreadyRegistered:
		return Failure{Status: http.StatusConflict, Code: "auth.google_email_registered", Message: "email already registered; sign in and link Google"}, true
	case apperrors.InvalidPhone:
		return Failure{Status: http.StatusBadRequest, Code: "auth.invalid_phone", Message: "invalid phone"}, true
	case apperrors.PhoneRegistered:
		return Failure{Status: http.StatusConflict, Code: "auth.phone_already_registered", Message: "phone number already registered"}, true
	case apperrors.PhoneUnavailable:
		return Failure{Status: http.StatusConflict, Code: "auth.phone_unavailable", Message: "phone unavailable"}, true
	case apperrors.OTPInvalid:
		return Failure{Status: http.StatusUnauthorized, Code: "auth.otp_invalid", Message: "invalid or expired code"}, true
	case apperrors.OTPCooldown:
		return Failure{Status: http.StatusTooManyRequests, Code: "auth.otp_cooldown", Message: "code temporarily unavailable"}, true
	default:
		return Failure{}, false
	}
}
