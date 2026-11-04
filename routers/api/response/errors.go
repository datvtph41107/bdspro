package response

import (
	"errors"
	account_model "github.com/datvtph41107/bdspro/models/account"
	auth_model "github.com/datvtph41107/bdspro/models/auth"
	organization_model "github.com/datvtph41107/bdspro/models/organization"
	"net/http"
)

// errorResponse contains the HTTP status and safe client message for a Go error.
type errorResponse struct {
	Status  int
	Code    string
	Message string
}

// getErrorResponse maps application errors to the existing HTTP contract.
func getErrorResponse(err error) (errorResponse, bool) {
	switch {
	case errors.Is(err, auth_model.ErrOAuth2CredentialAlreadyLinked):
		return errorResponse{Status: http.StatusConflict, Code: "oauth2.credential_already_linked", Message: "OAuth2 credential already linked"}, true
	case errors.Is(err, auth_model.ErrOAuth2EmailAlreadyRegistered):
		return errorResponse{Status: http.StatusConflict, Code: "oauth2.email_registered", Message: "email already registered; sign in to link"}, true
	case errors.Is(err, auth_model.ErrOAuth2EmailUnverified):
		return errorResponse{Status: http.StatusUnauthorized, Code: "oauth2.email_unverified", Message: "verified email required"}, true
	case errors.Is(err, auth_model.ErrOIDCIssuerInvalid), errors.Is(err, auth_model.ErrOIDCSubjectInvalid):
		return errorResponse{Status: http.StatusUnauthorized, Code: "oauth2.invalid_identity", Message: "invalid identity assertion"}, true
	case errors.Is(err, auth_model.ErrInvalidResetToken):
		return errorResponse{Status: http.StatusBadRequest, Code: "auth.invalid_reset_token", Message: "invalid or expired reset token"}, true
	case errors.Is(err, auth_model.ErrInvalidPassword):
		return errorResponse{Status: http.StatusBadRequest, Code: "auth.invalid_password", Message: "invalid request"}, true
	case errors.Is(err, account_model.ErrAccountUnavailable):
		return errorResponse{Status: http.StatusUnauthorized, Code: "auth.account_unavailable", Message: "account unavailable"}, true
	case errors.Is(err, organization_model.ErrNameRequired):
		return errorResponse{Status: http.StatusBadRequest, Code: "http.name_required", Message: "name required"}, true
	case errors.Is(err, organization_model.ErrInvalidOrganization):
		return errorResponse{Status: http.StatusBadRequest, Code: "http.invalid_organization", Message: "invalid organization"}, true
	case errors.Is(err, auth_model.ErrInvalidVerificationLink):
		return errorResponse{Status: http.StatusBadRequest, Code: "auth.invalid_verification_link", Message: "invalid or expired verification link"}, true
	case errors.Is(err, auth_model.ErrInvalidRefreshToken):
		return errorResponse{Status: http.StatusUnauthorized, Code: "auth.invalid_refresh_token", Message: "invalid refresh token"}, true
	case errors.Is(err, auth_model.ErrSessionNotFound):
		return errorResponse{Status: http.StatusNotFound, Code: "auth.session_not_found", Message: "session not found"}, true
	case errors.Is(err, auth_model.ErrSessionUnavailable):
		return errorResponse{Status: http.StatusUnauthorized, Code: "auth.session_unavailable", Message: "session unavailable"}, true
	case errors.Is(err, auth_model.ErrInvalidCredentials):
		return errorResponse{Status: http.StatusUnauthorized, Code: "auth.invalid_credentials", Message: "invalid credentials"}, true
	case errors.Is(err, auth_model.ErrInvalidEmail):
		return errorResponse{Status: http.StatusBadRequest, Code: "auth.invalid_email", Message: "invalid email"}, true
	case errors.Is(err, auth_model.ErrEmailAlreadyLinked):
		return errorResponse{Status: http.StatusConflict, Code: "auth.email_already_linked", Message: "account already has an email"}, true
	case errors.Is(err, auth_model.ErrInvalidEmailLink):
		return errorResponse{Status: http.StatusBadRequest, Code: "auth.invalid_email_link", Message: "invalid or expired email link"}, true
	case errors.Is(err, auth_model.ErrEmailLinkCooldown):
		return errorResponse{Status: http.StatusTooManyRequests, Code: "auth.email_link_cooldown", Message: "email verification temporarily unavailable"}, true
	case errors.Is(err, account_model.ErrEmailRegistered):
		return errorResponse{Status: http.StatusConflict, Code: "auth.email_registered", Message: "email already registered"}, true
	case errors.Is(err, organization_model.ErrOrganizationNotFound):
		return errorResponse{Status: http.StatusNotFound, Code: "organization.not_found", Message: "organization not found"}, true
	case errors.Is(err, organization_model.ErrOrganizationOwnerRequired):
		return errorResponse{Status: http.StatusForbidden, Code: "organization.owner_required", Message: "organization owner required"}, true
	case errors.Is(err, organization_model.ErrMembershipExists):
		return errorResponse{Status: http.StatusConflict, Code: "organization.membership_exists", Message: "membership already exists"}, true
	case errors.Is(err, organization_model.ErrPersonUnavailable):
		return errorResponse{Status: http.StatusNotFound, Code: "organization.person_unavailable", Message: "person unavailable"}, true
	case errors.Is(err, auth_model.ErrInvalidPhone):
		return errorResponse{Status: http.StatusBadRequest, Code: "auth.invalid_phone", Message: "invalid phone"}, true
	case errors.Is(err, auth_model.ErrPhoneRegistered):
		return errorResponse{Status: http.StatusConflict, Code: "auth.phone_already_registered", Message: "phone number already registered"}, true
	case errors.Is(err, auth_model.ErrPhoneUnavailable):
		return errorResponse{Status: http.StatusConflict, Code: "auth.phone_unavailable", Message: "phone unavailable"}, true
	case errors.Is(err, auth_model.ErrOTPInvalid):
		return errorResponse{Status: http.StatusUnauthorized, Code: "auth.otp_invalid", Message: "invalid or expired code"}, true
	case errors.Is(err, auth_model.ErrOTPCooldown):
		return errorResponse{Status: http.StatusTooManyRequests, Code: "auth.otp_cooldown", Message: "code temporarily unavailable"}, true
	case errors.Is(err, organization_model.ErrTeamNotFound):
		return errorResponse{Status: http.StatusNotFound, Code: "organization.team_not_found", Message: "team not found"}, true
	case errors.Is(err, organization_model.ErrTeamNameInvalid):
		return errorResponse{Status: http.StatusBadRequest, Code: "organization.team_name_invalid", Message: "invalid team name"}, true
	case errors.Is(err, organization_model.ErrTeamAlreadyExists):
		return errorResponse{Status: http.StatusConflict, Code: "organization.team_exists", Message: "team already exists"}, true
	case errors.Is(err, organization_model.ErrTeamOwnerRequired):
		return errorResponse{Status: http.StatusForbidden, Code: "organization.team_owner_required", Message: "organization owner required"}, true
	case errors.Is(err, organization_model.ErrTeamMemberExists):
		return errorResponse{Status: http.StatusConflict, Code: "organization.team_member_exists", Message: "team member already exists"}, true
	case errors.Is(err, organization_model.ErrTeamMemberNotFound):
		return errorResponse{Status: http.StatusNotFound, Code: "organization.team_member_not_found", Message: "team member not found"}, true
	case errors.Is(err, organization_model.ErrTeamPaginationInvalid):
		return errorResponse{Status: http.StatusBadRequest, Code: "organization.team_pagination_invalid", Message: "invalid team pagination"}, true
	case errors.Is(err, organization_model.ErrTeamInviteInvalid):
		return errorResponse{Status: http.StatusBadRequest, Code: "organization.team_invite_invalid", Message: "invalid or expired invitation"}, true
	case errors.Is(err, organization_model.ErrTeamInviteExists):
		return errorResponse{Status: http.StatusConflict, Code: "organization.team_invite_exists", Message: "team invitation already exists"}, true
	default:
		return errorResponse{}, false
	}
}
