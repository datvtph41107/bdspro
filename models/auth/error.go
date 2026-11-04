package auth

import (
	"errors"

	"github.com/datvtph41107/bdspro/modules/util"
)

// Authentication model errors have no HTTP status or public message contract.
var (
	ErrPasswordResetInvalid          = errors.New("invalid or expired password reset")
	ErrInvalidResetToken             = util.WithPublic(util.ErrInvalidArgument, "invalid reset token", "auth.invalid_reset_token", "invalid or expired reset token")
	ErrInvalidPassword               = util.WithPublic(util.ErrInvalidArgument, "invalid password", "auth.invalid_password", "invalid request")
	ErrInvalidVerificationLink       = util.WithPublic(util.ErrInvalidArgument, "invalid verification link", "auth.invalid_verification_link", "invalid or expired verification link")
	ErrInvalidRefreshToken           = util.WithPublic(util.ErrUnauthenticated, "invalid refresh token", "auth.invalid_refresh_token", "invalid refresh token")
	ErrSessionNotFound               = util.WithPublic(util.ErrNotExist, "session not found", "auth.session_not_found", "session not found")
	ErrSessionUnavailable            = util.WithPublic(util.ErrUnauthenticated, "session unavailable", "auth.session_unavailable", "session unavailable")
	ErrInvalidCredentials            = util.WithPublic(util.ErrUnauthenticated, "invalid credentials", "auth.invalid_credentials", "invalid credentials")
	ErrInvalidEmail                  = util.WithPublic(util.ErrInvalidArgument, "invalid email", "auth.invalid_email", "invalid email")
	ErrEmailAlreadyLinked            = util.WithPublic(util.ErrAlreadyExist, "email already linked", "auth.email_already_linked", "account already has an email")
	ErrInvalidEmailLink              = util.WithPublic(util.ErrInvalidArgument, "invalid email link", "auth.invalid_email_link", "invalid or expired email link")
	ErrEmailLinkCooldown             = util.WithPublic(util.ErrRateLimited, "email link cooldown", "auth.email_link_cooldown", "email verification temporarily unavailable")
	ErrInvalidPhone                  = util.WithPublic(util.ErrInvalidArgument, "invalid phone", "auth.invalid_phone", "invalid phone")
	ErrPhoneRegistered               = util.WithPublic(util.ErrAlreadyExist, "phone already registered", "auth.phone_already_registered", "phone number already registered")
	ErrPhoneUnavailable              = util.WithPublic(util.ErrAlreadyExist, "phone unavailable", "auth.phone_unavailable", "phone unavailable")
	ErrOTPInvalid                    = util.WithPublic(util.ErrUnauthenticated, "invalid or expired one-time code", "auth.otp_invalid", "invalid or expired code")
	ErrOTPCooldown                   = util.WithPublic(util.ErrRateLimited, "one-time code cooldown", "auth.otp_cooldown", "code temporarily unavailable")
	ErrOIDCIssuerInvalid             = util.WithPublic(util.ErrUnauthenticated, "invalid OIDC issuer", "oauth2.invalid_identity", "invalid identity assertion")
	ErrOIDCSubjectInvalid            = util.WithPublic(util.ErrUnauthenticated, "invalid OIDC subject", "oauth2.invalid_identity", "invalid identity assertion")
	ErrOAuth2CredentialAlreadyLinked = util.WithPublic(util.ErrAlreadyExist, "external identity already linked", "oauth2.credential_already_linked", "OAuth2 credential already linked")
	ErrOAuth2EmailUnverified         = util.WithPublic(util.ErrUnauthenticated, "external identity email unverified", "oauth2.email_unverified", "verified email required")
	ErrOAuth2EmailAlreadyRegistered  = util.WithPublic(util.ErrAlreadyExist, "email already registered; reauthentication required", "oauth2.email_registered", "email already registered; sign in to link")
)
