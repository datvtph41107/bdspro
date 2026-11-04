package auth

import (
	"errors"
	"net/url"
	"strings"
)

var (
	ErrOIDCIssuerInvalid  = errors.New("invalid OIDC issuer")
	ErrOIDCSubjectInvalid           = errors.New("invalid OIDC subject")
	ErrOAuth2IdentityAlreadyLinked  = errors.New("external identity already linked")
	ErrOAuth2EmailUnverified        = errors.New("external identity email unverified")
	ErrOAuth2EmailAlreadyRegistered = errors.New("email already registered; reauthentication required")
)

// OIDCIdentity carries identity claims AFTER an OpenID Connect ID token was
// cryptographically verified against a trusted provider configuration.
// This value type does not verify or grant account access on its own.
type OIDCIdentity struct {
	Issuer        string
	Subject       string
	Email         string
	EmailVerified bool
}

// IdentityKey is globally unique within the configured set of trusted issuers.
// Never use email as an identity key or silently link identities by email.
type IdentityKey struct {
	Issuer  string
	Subject string
}

// Key checks key shape only; it does not validate signature, audience or nonce.
// Preserve the verified issuer exactly, including its path when present.
func (identity OIDCIdentity) Key() (IdentityKey, error) {
	issuer := identity.Issuer
	parsed, err := url.Parse(issuer)
	if err != nil || parsed == nil || parsed.Scheme != "https" || parsed.Host == "" ||
		parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" ||
		parsed.Opaque != "" || parsed.String() != issuer {
		return IdentityKey{}, ErrOIDCIssuerInvalid
	}
	if strings.TrimSpace(identity.Subject) == "" || identity.Subject != strings.TrimSpace(identity.Subject) {
		return IdentityKey{}, ErrOIDCSubjectInvalid
	}
	return IdentityKey{Issuer: issuer, Subject: identity.Subject}, nil
}
