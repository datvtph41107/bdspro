package auth

import (
	"context"
	"net/http"

	auth_model "github.com/datvtph41107/bdspro/models/auth"
)

const googleJWKSURL = "https://www.googleapis.com/oauth2/v3/certs"

// Legacy compatibility only. New authentication integrations must use the
// provider-independent OIDCTokenVerifier and OIDCClaims.
type VerifiedGoogleIdentity = auth_model.OIDCClaims
type googleIDClaims = oidcIDClaims

func VerifyGoogleIDToken(ctx context.Context, raw, clientID, nonce string) (VerifiedGoogleIdentity, error) {
	return verifyGoogleIDToken(ctx, http.DefaultClient, googleJWKSURL, raw, clientID, nonce)
}

func verifyGoogleIDToken(ctx context.Context, client *http.Client, jwksURL, raw, clientID, nonce string) (VerifiedGoogleIdentity, error) {
	return (OIDCTokenVerifier{
		Issuer: "https://accounts.google.com", Audience: clientID,
		JWKSURL: jwksURL, Client: client,
	}).Verify(ctx, raw, nonce)
}
