package apperrors

import (
	"errors"
	"net/http"
)

// Error is a public, immutable-by-convention application failure.
// Code identifies a stable failure; Message is safe for clients.
// Cause is internal and is never serialized by the HTTP writer.
type Error struct {
	Code    string
	Message string
	Status  int
	Cause   error
}

func (e *Error) Error() string { return e.Message }
func (e *Error) Unwrap() error { return e.Cause }

// Wrap preserves a canonical error definition while retaining its underlying cause.
func Wrap(def *Error, cause error) error {
	if cause == nil {
		return def
	}
	return &Error{Code: def.Code, Message: def.Message, Status: def.Status, Cause: cause}
}

func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	return ok && e.Code != "" && e.Code == t.Code
}

// Business errors and infrastructure errors are distinct.
// Internal errors deliberately have no client-facing status/message.
var (
	InvalidPassword           = &Error{Code: "auth.invalid_password", Message: "invalid request", Status: http.StatusBadRequest}
	AccountUnavailable        = &Error{Code: "auth.account_unavailable", Message: "account unavailable", Status: http.StatusUnauthorized}
	NameRequired              = &Error{Code: "http.name_required", Message: "name required", Status: http.StatusBadRequest}
	InvalidOrganization       = &Error{Code: "http.invalid_organization", Message: "invalid organization", Status: http.StatusBadRequest}
	InvalidVerificationLink   = &Error{Code: "auth.invalid_verification_link", Message: "invalid or expired verification link", Status: http.StatusBadRequest}
	InvalidRefreshToken       = &Error{Code: "auth.invalid_refresh_token", Message: "invalid refresh token", Status: http.StatusUnauthorized}
	SessionUnavailable        = &Error{Code: "auth.session_unavailable", Message: "session unavailable", Status: http.StatusUnauthorized}
	InvalidCredentials        = &Error{Code: "auth.invalid_credentials", Message: "invalid credentials", Status: http.StatusUnauthorized}
	EmailRegistered           = &Error{Code: "auth.email_registered", Message: "email already registered", Status: http.StatusConflict}
	OrganizationOwnerRequired = &Error{Code: "organization.owner_required", Message: "organization owner required", Status: http.StatusForbidden}
	MembershipExists          = &Error{Code: "organization.membership_exists", Message: "membership already exists", Status: http.StatusConflict}
	PersonUnavailable         = &Error{Code: "organization.person_unavailable", Message: "person unavailable", Status: http.StatusNotFound}
	InvalidPhone              = &Error{Code: "auth.invalid_phone", Message: "invalid phone", Status: http.StatusBadRequest}
	PhoneUnavailable          = &Error{Code: "auth.phone_unavailable", Message: "phone unavailable", Status: http.StatusConflict}
	OTPInvalid                = &Error{Code: "auth.otp_invalid", Message: "invalid or expired code", Status: http.StatusUnauthorized}
	OTPCooldown               = &Error{Code: "auth.otp_cooldown", Message: "code temporarily unavailable", Status: http.StatusTooManyRequests}
)

func Public(err error) (*Error, bool) {
	var e *Error
	if !errors.As(err, &e) || e.Status < 400 || e.Status >= 500 {
		return nil, false
	}
	return e, true
}
