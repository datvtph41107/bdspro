package routers

import (
	"encoding/json"
	"testing"

	"github.com/datvtph41107/bdspro/templates"
)

func TestOAuthConsentVersionOpenAPI(t *testing.T) {
	raw, err := templates.Swagger.ReadFile("swagger/openapi.json")
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Paths map[string]map[string]json.RawMessage `json:"paths"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	var preview struct {
		Responses map[string]struct {
			Content map[string]struct {
				Schema struct {
					Properties map[string]json.RawMessage `json:"properties"`
				} `json:"schema"`
			} `json:"content"`
		} `json:"responses"`
	}
	if err := json.Unmarshal(doc.Paths["/oauth/authorize"]["get"], &preview); err != nil {
		t.Fatal(err)
	}
	if _, ok := preview.Responses["200"].Content["application/json"].Schema.Properties["consent_version"]; !ok {
		t.Fatal("consent preview must expose version")
	}
	var approval struct {
		RequestBody struct {
			Content map[string]struct {
				Schema struct {
					Properties map[string]json.RawMessage `json:"properties"`
				} `json:"schema"`
			} `json:"content"`
		} `json:"requestBody"`
	}
	if err := json.Unmarshal(doc.Paths["/oauth/authorize"]["post"], &approval); err != nil {
		t.Fatal(err)
	}
	props := approval.RequestBody.Content["application/x-www-form-urlencoded"].Schema.Properties
	if _, ok := props["consent_version"]; !ok {
		t.Fatal("approval omits required consent_version parameter")
	}
}
