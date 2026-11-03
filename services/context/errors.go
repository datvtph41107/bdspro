package context

import (
	"errors"
	"net/http"

	apperrors "github.com/datvtph41107/bdspro/modules/errors"
)

// applicationError contains the identical 29 public status/code/message
// mappings formerly owned by routers/api/response/errors.go. Do not introduce
// a second mapping at HTTP cutover; remove the Gin mapping only after parity.
func applicationError(err error) (status int, code, message string, ok bool) {
	var app *apperrors.Error
	if !errors.As(err, &app) {
		return 0, "", "", false
	}
	switch app {
	case apperrors.InvalidResetToken:
		return http.StatusBadRequest, "auth.invalid_reset_token", "invalid or expired reset token", true
	case apperrors.InvalidPassword:
		return http.StatusBadRequest, "auth.invalid_password", "invalid request", true
	case apperrors.AccountUnavailable:
		return http.StatusUnauthorized, "auth.account_unavailable", "account unavailable", true
	case apperrors.NameRequired:
		return http.StatusBadRequest, "http.name_required", "name required", true
	case apperrors.InvalidOrganization:
		return http.StatusBadRequest, "http.invalid_organization", "invalid organization", true
	case apperrors.InvalidVerificationLink:
		return http.StatusBadRequest, "auth.invalid_verification_link", "invalid or expired verification link", true
	case apperrors.InvalidRefreshToken:
		return http.StatusUnauthorized, "auth.invalid_refresh_token", "invalid refresh token", true
	case apperrors.SessionNotFound:
		return http.StatusNotFound, "auth.session_not_found", "session not found", true
	case apperrors.SessionUnavailable:
		return http.StatusUnauthorized, "auth.session_unavailable", "session unavailable", true
	case apperrors.InvalidCredentials:
		return http.StatusUnauthorized, "auth.invalid_credentials", "invalid credentials", true
	case apperrors.InvalidEmail:
		return http.StatusBadRequest, "auth.invalid_email", "invalid email", true
	case apperrors.EmailAlreadyLinked:
		return http.StatusConflict, "auth.email_already_linked", "account already has an email", true
	case apperrors.InvalidEmailLink:
		return http.StatusBadRequest, "auth.invalid_email_link", "invalid or expired email link", true
	case apperrors.EmailLinkCooldown:
		return http.StatusTooManyRequests, "auth.email_link_cooldown", "email verification temporarily unavailable", true
	case apperrors.EmailRegistered:
		return http.StatusConflict, "auth.email_registered", "email already registered", true
	case apperrors.OrganizationNotFound:
		return http.StatusNotFound, "organization.not_found", "organization not found", true
	case apperrors.OrganizationOwnerRequired:
		return http.StatusForbidden, "organization.owner_required", "organization owner required", true
	case apperrors.MembershipExists:
		return http.StatusConflict, "organization.membership_exists", "membership already exists", true
	case apperrors.PersonUnavailable:
		return http.StatusNotFound, "organization.person_unavailable", "person unavailable", true
	case apperrors.GoogleIdentityAlreadyLinked:
		return http.StatusConflict, "auth.google_identity_already_linked", "Google identity already linked", true
	case apperrors.GoogleAccountAlreadyLinked:
		return http.StatusConflict, "auth.google_account_already_linked", "account already linked", true
	case apperrors.GoogleIdentityUnlinked:
		return http.StatusUnauthorized, "auth.google_identity_unlinked", "Google identity not linked", true
	case apperrors.GoogleEmailUnverified:
		return http.StatusUnauthorized, "auth.google_email_unverified", "Google email is not verified", true
	case apperrors.GoogleEmailAlreadyRegistered:
		return http.StatusConflict, "auth.google_email_registered", "email already registered; sign in and link Google", true
	case apperrors.InvalidPhone:
		return http.StatusBadRequest, "auth.invalid_phone", "invalid phone", true
	case apperrors.PhoneRegistered:
		return http.StatusConflict, "auth.phone_already_registered", "phone number already registered", true
	case apperrors.PhoneUnavailable:
		return http.StatusConflict, "auth.phone_unavailable", "phone unavailable", true
	case apperrors.OTPInvalid:
		return http.StatusUnauthorized, "auth.otp_invalid", "invalid or expired code", true
	case apperrors.OTPCooldown:
		return http.StatusTooManyRequests, "auth.otp_cooldown", "code temporarily unavailable", true
	default:
		return 0, "", "", false
	}
}
