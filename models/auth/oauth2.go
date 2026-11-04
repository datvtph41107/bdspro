package auth

import (
    "errors"
    "net/url"
    "strings"
)

var (
    ErrOIDCIssuerInvalid = errors.New("invalid OIDC issuer")
    ErrOIDCSubjectInvalid = errors.New("invalid OIDC subject")
)

// OIDCIdentity contains claims from a verified OpenID Connect ID token.
// Constructing it does NOT verify a token. Callers must first check signature,
// issuer, audience, expiry, nonce and other required claims using provider
// metadata. Email must not be used as an identity key for account linking.
type OIDCIdentity struct {
    Issuer string
    Subject string
    Email string
    EmailVerified bool
}

// IdentityKey is scoped by the OpenID Provider issuer, not email.
type IdentityKey struct {
    Issuer string
    Subject string
}

// Key checks the shape of a verified issuer and subject. It is NOT token
// verification. Preserve the issuer exactly as verified against configuration.
func (identity OIDCIdentity) Key() (IdentityKey, error) {
    issuer := identity.Issuer
    parsed, err := url.Parse(issuer)
    if err != nil || parsed == nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Opaque != "" || parsed.String() != issuer {
        return IdentityKey{}, ErrOIDCIssuerInvalid
    }
    if strings.TrimSpace(identity.Subject) == "" || identity.Subject != strings.TrimSpace(identity.Subject) {
        return IdentityKey{}, ErrOIDCSubjectInvalid
    }
    return IdentityKey{Issuer: issuer, Subject: identity.Subject}, nil
}
