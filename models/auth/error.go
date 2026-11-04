package auth

import (
	"errors"

	"github.com/datvtph41107/bdspro/modules/util"
)

// Authentication model errors have no HTTP status or public message contract.
var (
	ErrPasswordResetInvalid          = errors.New("invalid or expired password reset")
	ErrInvalidResetToken             = errors.New("invalid reset token")
	ErrInvalidPassword               = errors.New("invalid password")
	ErrInvalidVerificationLink       = errors.New("invalid verification link")
	ErrInvalidRefreshToken           = errors.New("invalid refresh token")
	ErrSessionNotFound               = util.WithKind(util.ErrNotExist, "session not found")
	ErrSessionUnavailable            = errors.New("session unavailable")
	ErrInvalidCredentials            = errors.New("invalid credentials")
	ErrInvalidEmail                  = errors.New("invalid email")
	ErrEmailAlreadyLinked            = errors.New("email already linked")
	ErrInvalidEmailLink              = errors.New("invalid email link")
	ErrEmailLinkCooldown             = errors.New("email link cooldown")
	ErrInvalidPhone                  = errors.New("invalid phone")
	ErrPhoneRegistered               = errors.New("phone already registered")
	ErrPhoneUnavailable              = errors.New("phone unavailable")
	ErrOTPInvalid                    = errors.New("invalid or expired one-time code")
	ErrOTPCooldown                   = errors.New("one-time code cooldown")
	ErrOIDCIssuerInvalid             = errors.New("invalid OIDC issuer")
	ErrOIDCSubjectInvalid            = errors.New("invalid OIDC subject")
	ErrOAuth2CredentialAlreadyLinked = util.WithKind(util.ErrAlreadyExist, "external identity already linked")
	ErrOAuth2EmailUnverified         = errors.New("external identity email unverified")
	ErrOAuth2EmailAlreadyRegistered  = util.WithKind(util.ErrAlreadyExist, "email already registered; reauthentication required")
)
