package auth

import (
	"bytes"
	legacyjson "encoding/json"
	"testing"

	appjson "github.com/datvtph41107/bdspro/modules/json"
)

// OAuth2 state already stored in Redis must remain readable during deploy.
func TestOAuth2PendingJSONCompatibility(t *testing.T) {
	values := []OAuth2Pending{
		{Issuer: "https://issuer.example", Purpose: "login", Verifier: "verifier", Nonce: "nonce"},
		{Issuer: "https://issuer.example", Purpose: "phone_bind", AccountID: 12, Verifier: "verifier", Nonce: "nonce", Phone: "+84912345678"},
	}
	for _, value := range values {
		oldJSON, err := legacyjson.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		newJSON, err := appjson.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(oldJSON, newJSON) {
			t.Fatalf("OIDC pending state changed: old=%s new=%s", oldJSON, newJSON)
		}
		var decoded OAuth2Pending
		if err := appjson.Unmarshal(oldJSON, &decoded); err != nil {
			t.Fatalf("old OAuth2 pending state rejected: %v", err)
		}
		if decoded != value {
			t.Fatalf("state mismatch: got=%+v want=%+v", decoded, value)
		}
	}
}

func TestOIDCProviderJSONRejectsDuplicateKey(t *testing.T) {
	var response struct {
		IDToken string `json:"id_token"`
	}
	if err := appjson.Unmarshal([]byte(`{"id_token":"trusted","id_token":"untrusted"}`), &response); err == nil {
		t.Fatal("duplicate OIDC token names accepted")
	}
}
