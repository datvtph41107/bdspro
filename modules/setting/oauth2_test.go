package setting

import "testing"

func TestOAuth2ProviderValidation(t *testing.T) {
	valid := OAuth2Provider{
		Issuer:           "https://issuer.example/tenant",
		AuthorizationURL: "https://issuer.example/authorize",
		TokenURL:         "https://issuer.example/token",
		JWKSURL:          "https://issuer.example/keys",
		ClientID:         "test-client", ClientSecret: "secret",
		RedirectURI:      "http://localhost:8080/users/oauth2/callback",
	}
	if err := valid.Validate(); err != nil {
		t.Fatal(err)
	}
	invalid := valid
	invalid.JWKSURL = "http://attacker.example/keys"
	if err := invalid.Validate(); err == nil {
		t.Fatal("insecure JWKS endpoint accepted")
	}
	invalid = valid
	invalid.RedirectURI = "https://example.test/callback#fragment"
	if err := invalid.Validate(); err == nil {
		t.Fatal("fragment-bearing redirect accepted")
	}
}
