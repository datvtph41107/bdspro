package apperrors

import "errors"

// Error is an application failure identity, independent of HTTP status,
// response codes, and client-facing messages.
type Error struct {
	name string
}

func (e *Error) Error() string { return e.name }

// Wrap retains both the application identity and the underlying cause.
// Callers can inspect either with errors.Is/errors.As.
func Wrap(def *Error, cause error) error {
	if cause == nil {
		return def
	}
	return errors.Join(def, cause)
}

// These identities are independent of the public API. Public error
// mappings live only in internal/api/response.
var InvalidResetToken = &Error{name: "InvalidResetToken"}
var InvalidPassword = &Error{name: "InvalidPassword"}
var AccountUnavailable = &Error{name: "AccountUnavailable"}
var NameRequired = &Error{name: "NameRequired"}
var InvalidOrganization = &Error{name: "InvalidOrganization"}
var InvalidVerificationLink = &Error{name: "InvalidVerificationLink"}
var InvalidRefreshToken = &Error{name: "InvalidRefreshToken"}
var SessionNotFound = &Error{name: "SessionNotFound"}
var SessionUnavailable = &Error{name: "SessionUnavailable"}
var InvalidCredentials = &Error{name: "InvalidCredentials"}
var InvalidEmail = &Error{name: "InvalidEmail"}
var EmailAlreadyLinked = &Error{name: "EmailAlreadyLinked"}
var InvalidEmailLink = &Error{name: "InvalidEmailLink"}
var EmailLinkCooldown = &Error{name: "EmailLinkCooldown"}
var EmailRegistered = &Error{name: "EmailRegistered"}
var OrganizationNotFound = &Error{name: "OrganizationNotFound"}
var OrganizationOwnerRequired = &Error{name: "OrganizationOwnerRequired"}
var MembershipExists = &Error{name: "MembershipExists"}
var PersonUnavailable = &Error{name: "PersonUnavailable"}
var GoogleIdentityAlreadyLinked = &Error{name: "GoogleIdentityAlreadyLinked"}
var GoogleAccountAlreadyLinked = &Error{name: "GoogleAccountAlreadyLinked"}
var GoogleIdentityUnlinked = &Error{name: "GoogleIdentityUnlinked"}
var GoogleEmailUnverified = &Error{name: "GoogleEmailUnverified"}
var GoogleEmailAlreadyRegistered = &Error{name: "GoogleEmailAlreadyRegistered"}
var InvalidPhone = &Error{name: "InvalidPhone"}
var PhoneRegistered = &Error{name: "PhoneRegistered"}
var PhoneUnavailable = &Error{name: "PhoneUnavailable"}
var OTPInvalid = &Error{name: "OTPInvalid"}
var OTPCooldown = &Error{name: "OTPCooldown"}
var InvalidProfile = &Error{name: "InvalidProfile"}
var InvalidSettings = &Error{name: "InvalidSettings"}
