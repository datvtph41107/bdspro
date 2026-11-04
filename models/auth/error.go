package auth

import "errors"

// Authentication model errors have no HTTP status or public message contract.
var (
	ErrPasswordResetInvalid         = errors.New("invalid or expired password reset")
	ErrInvalidResetToken            = errors.New("invalid reset token")
	ErrInvalidPassword              = errors.New("invalid password")
	ErrInvalidVerificationLink      = errors.New("invalid verification link")
	ErrInvalidRefreshToken          = errors.New("invalid refresh token")
	ErrSessionNotFound              = errors.New("session not found")
	ErrSessionUnavailable           = errors.New("session unavailable")
	ErrInvalidCredentials           = errors.New("invalid credentials")
	ErrInvalidEmail                 = errors.New("invalid email")
	ErrEmailAlreadyLinked           = errors.New("email already linked")
	ErrInvalidEmailLink             = errors.New("invalid email link")
	ErrEmailLinkCooldown            = errors.New("email link cooldown")
	ErrGoogleIdentityAlreadyLinked  = errors.New("google identity already linked")
	ErrGoogleAccountAlreadyLinked   = errors.New("google account already linked")
	ErrGoogleIdentityUnlinked       = errors.New("google identity unlinked")
	ErrGoogleEmailUnverified        = errors.New("google email unverified")
	ErrGoogleEmailAlreadyRegistered = errors.New("google email already registered")
	ErrInvalidPhone                 = errors.New("invalid phone")
	ErrPhoneRegistered              = errors.New("phone already registered")
	ErrPhoneUnavailable             = errors.New("phone unavailable")
	ErrOTPInvalid                   = errors.New("invalid or expired one-time code")
	ErrOTPCooldown                  = errors.New("one-time code cooldown")
)
