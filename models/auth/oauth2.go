package auth

import (
	"net/url"
	"strings"
)

// OIDCClaims carries identity claims AFTER an OpenID Connect ID token was
// cryptographically verified against a trusted provider configuration.
// This value type does not verify or grant account access on its own.
type OIDCClaims struct {
	Issuer        string
	Subject       string
	Email         string
	EmailVerified bool
}

// Key checks key shape only; it does not validate signature, audience or nonce.
// Preserve the verified issuer exactly, including its path when present.
func (identity OIDCClaims) Key() (OAuth2CredentialKey, error) {
	issuer := identity.Issuer
	parsed, err := url.Parse(issuer)
	if err != nil || parsed == nil || parsed.Scheme != "https" || parsed.Host == "" ||
		parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" ||
		parsed.Opaque != "" || parsed.String() != issuer {
		return OAuth2CredentialKey{}, ErrOIDCIssuerInvalid
	}
	if len(identity.Subject) > 255 {
		return OAuth2CredentialKey{}, ErrOIDCSubjectInvalid
	}
	for i := 0; i < len(identity.Subject); i++ {
		if identity.Subject[i] > 127 {
			return OAuth2CredentialKey{}, ErrOIDCSubjectInvalid
		}
	}
	if strings.TrimSpace(identity.Subject) == "" || identity.Subject != strings.TrimSpace(identity.Subject) {
		return OAuth2CredentialKey{}, ErrOIDCSubjectInvalid
	}
	return OAuth2CredentialKey{Issuer: issuer, Subject: identity.Subject}, nil
}
